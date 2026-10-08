package mbsrver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

const (
	functionReadCoils            = 1
	functionReadDiscreteInputs   = 2
	functionReadHoldingRegisters = 3
	functionReadInputRegisters   = 4
	functionWriteSingleCoil      = 5
	functionWriteSingleRegister  = 6
	functionWriteMultipleCoils   = 15
	functionWriteMultipleRegs    = 16

	exceptionIllegalFunction    = 1
	exceptionIllegalDataAddress = 2
	exceptionIllegalDataValue   = 3
)

type modbusServer struct {
	id        string
	port      int
	slaveID   int
	registers []*register
	index     map[registerTable]map[int]*register
	handlers  []trigger.Handler
	logger    log.Logger

	mu       sync.Mutex
	listener net.Listener
	conns    map[net.Conn]struct{}
	started  bool
	ctx      context.Context
	cancel   context.CancelFunc
	wait     sync.WaitGroup

	valuesMu sync.RWMutex
	holding  map[int]uint16
	input    map[int]uint16
	coils    map[int]bool
	discrete map[int]bool
}

func newModbusServer(id string, settings *Settings) (*modbusServer, error) {
	if err := validateSettings(settings); err != nil {
		return nil, err
	}
	registers, err := parseRegisters(settings.Registers)
	if err != nil {
		return nil, err
	}

	server := &modbusServer{
		id:        id,
		port:      settings.Port,
		slaveID:   settings.SlaveID,
		registers: registers,
		index:     indexRegisters(registers),
		conns:     make(map[net.Conn]struct{}),
		holding:   make(map[int]uint16),
		input:     make(map[int]uint16),
		coils:     make(map[int]bool),
		discrete:  make(map[int]bool),
	}
	server.initializeValues()
	return server, nil
}

func (s *modbusServer) initialize(logger log.Logger, handlers []trigger.Handler) {
	s.logger = logger
	s.handlers = handlers
}

func (s *modbusServer) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		return fmt.Errorf("starting Modbus TCP server on port %d: %w", s.port, err)
	}
	s.listener = listener
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.started = true
	s.wait.Add(1)
	go s.serve(listener)
	return nil
}

func (s *modbusServer) stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	s.cancel()
	listener := s.listener
	connections := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		connections = append(connections, conn)
	}
	s.mu.Unlock()

	var closeErr error
	if err := listener.Close(); err != nil && !isClosedNetworkError(err) {
		closeErr = err
	}
	for _, conn := range connections {
		_ = conn.Close()
	}
	s.wait.Wait()
	if closeErr != nil {
		return fmt.Errorf("stopping Modbus TCP server: %w", closeErr)
	}
	return nil
}

func (s *modbusServer) serve(listener net.Listener) {
	defer s.wait.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !isClosedNetworkError(err) {
				s.logger.Errorf("Modbus server %q accept failed: %v", s.id, err)
			}
			return
		}

		s.mu.Lock()
		if !s.started {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		s.conns[conn] = struct{}{}
		s.wait.Add(1)
		s.mu.Unlock()
		go s.serveConnection(conn)
	}
}

func (s *modbusServer) serveConnection(conn net.Conn) {
	defer s.wait.Done()
	defer func() {
		_ = conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	for {
		header := make([]byte, 7)
		if _, err := io.ReadFull(conn, header); err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				s.logger.Errorf("Modbus server %q read header failed: %v", s.id, err)
			}
			return
		}
		length := int(binary.BigEndian.Uint16(header[4:6]))
		if length < 2 || length > 254 {
			s.logger.Errorf("Modbus server %q received invalid frame length %d", s.id, length)
			return
		}
		pdu := make([]byte, length-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			s.logger.Errorf("Modbus server %q read request failed: %v", s.id, err)
			return
		}
		if binary.BigEndian.Uint16(header[2:4]) != 0 || int(header[6]) != s.slaveID {
			continue
		}

		response, events := s.processPDU(pdu)
		frame := make([]byte, 7, 7+len(response))
		copy(frame, header)
		binary.BigEndian.PutUint16(frame[4:6], uint16(len(response)+1))
		frame = append(frame, response...)
		if _, err := conn.Write(frame); err != nil {
			s.logger.Errorf("Modbus server %q write response failed: %v", s.id, err)
			return
		}
		for _, event := range events {
			s.emit(event)
		}
	}
}

func (s *modbusServer) processPDU(pdu []byte) ([]byte, []map[string]interface{}) {
	if len(pdu) == 0 {
		return exceptionResponse(0, exceptionIllegalDataValue), nil
	}
	function := int(pdu[0])
	switch function {
	case functionReadCoils, functionReadDiscreteInputs, functionReadHoldingRegisters, functionReadInputRegisters:
		return s.readPDU(function, pdu)
	case functionWriteSingleCoil, functionWriteSingleRegister, functionWriteMultipleCoils, functionWriteMultipleRegs:
		return s.writePDU(function, pdu)
	default:
		return exceptionResponse(function, exceptionIllegalFunction), nil
	}
}

func (s *modbusServer) readPDU(function int, pdu []byte) ([]byte, []map[string]interface{}) {
	if len(pdu) != 5 {
		return exceptionResponse(function, exceptionIllegalDataValue), nil
	}
	address := int(binary.BigEndian.Uint16(pdu[1:3]))
	count := int(binary.BigEndian.Uint16(pdu[3:5]))
	maxCount := 125
	if function == functionReadCoils || function == functionReadDiscreteInputs {
		maxCount = 2000
	}
	if count < 1 || count > maxCount {
		return exceptionResponse(function, exceptionIllegalDataValue), nil
	}

	s.valuesMu.RLock()
	defer s.valuesMu.RUnlock()
	table := tableForFunction(function)
	if !s.rangeConfigured(table, address, count) {
		return exceptionResponse(function, exceptionIllegalDataAddress), nil
	}
	if table == tableCoils || table == tableDiscrete {
		dataSize := (count + 7) / 8
		response := make([]byte, 2+dataSize)
		response[0] = byte(function)
		response[1] = byte(dataSize)
		values := s.coils
		if table == tableDiscrete {
			values = s.discrete
		}
		for index := 0; index < count; index++ {
			if values[address+index] {
				response[2+index/8] |= 1 << uint(index%8)
			}
		}
		return response, nil
	}

	response := make([]byte, 2+count*2)
	response[0] = byte(function)
	response[1] = byte(count * 2)
	values := s.holding
	if table == tableInput {
		values = s.input
	}
	for index := 0; index < count; index++ {
		binary.BigEndian.PutUint16(response[2+index*2:], values[address+index])
	}
	return response, nil
}

func (s *modbusServer) writePDU(function int, pdu []byte) ([]byte, []map[string]interface{}) {
	if (function == functionWriteSingleCoil || function == functionWriteSingleRegister) && len(pdu) != 5 {
		return exceptionResponse(function, exceptionIllegalDataValue), nil
	}
	if (function == functionWriteMultipleCoils || function == functionWriteMultipleRegs) && len(pdu) < 6 {
		return exceptionResponse(function, exceptionIllegalDataValue), nil
	}

	address := int(binary.BigEndian.Uint16(pdu[1:3]))
	count := 1
	if function == functionWriteMultipleCoils || function == functionWriteMultipleRegs {
		count = int(binary.BigEndian.Uint16(pdu[3:5]))
	}
	isCoilWrite := function == functionWriteSingleCoil || function == functionWriteMultipleCoils
	table := tableHolding
	maxCount := 123
	if isCoilWrite {
		table = tableCoils
		maxCount = 1968
	}
	if count < 1 || count > maxCount {
		return exceptionResponse(function, exceptionIllegalDataValue), nil
	}
	if address+count > 65536 {
		return exceptionResponse(function, exceptionIllegalDataAddress), nil
	}

	values := make([]uint16, count)
	bits := make([]bool, count)
	switch function {
	case functionWriteSingleCoil:
		value := binary.BigEndian.Uint16(pdu[3:5])
		if value != 0 && value != 0xff00 {
			return exceptionResponse(function, exceptionIllegalDataValue), nil
		}
		bits[0] = value == 0xff00
	case functionWriteSingleRegister:
		values[0] = binary.BigEndian.Uint16(pdu[3:5])
	case functionWriteMultipleCoils:
		byteCount := int(pdu[5])
		if byteCount != (count+7)/8 || len(pdu) != 6+byteCount {
			return exceptionResponse(function, exceptionIllegalDataValue), nil
		}
		for index := range bits {
			bits[index] = pdu[6+index/8]&(1<<uint(index%8)) != 0
		}
	case functionWriteMultipleRegs:
		byteCount := int(pdu[5])
		if byteCount != count*2 || len(pdu) != 6+byteCount {
			return exceptionResponse(function, exceptionIllegalDataValue), nil
		}
		for index := range values {
			values[index] = binary.BigEndian.Uint16(pdu[6+index*2:])
		}
	}

	s.valuesMu.Lock()
	if !s.rangeConfigured(table, address, count) {
		s.valuesMu.Unlock()
		return exceptionResponse(function, exceptionIllegalDataAddress), nil
	}
	affected := s.affectedRegisters(table, address, count)
	before := make(map[*register]interface{}, len(affected))
	for _, reg := range affected {
		before[reg] = s.registerValue(reg)
	}
	if isCoilWrite {
		for index, value := range bits {
			s.coils[address+index] = value
		}
	} else {
		for index, value := range values {
			s.holding[address+index] = value
		}
	}
	events := make([]map[string]interface{}, 0, len(affected))
	for _, reg := range affected {
		value := s.registerValue(reg)
		if !reflect.DeepEqual(before[reg], value) {
			events = append(events, reg.event(value))
		}
	}
	s.valuesMu.Unlock()

	response := append([]byte{byte(function)}, pdu[1:5]...)
	return response, events
}

func (s *modbusServer) emit(event map[string]interface{}) {
	for _, handler := range s.handlers {
		if _, err := handler.Handle(s.ctx, (&Output{Data: event}).ToMap()); err != nil {
			s.logger.Errorf("Modbus server handler %q failed: %v", handler.Name(), err)
		}
	}
}

func (s *modbusServer) initializeValues() {
	for _, reg := range s.registers {
		switch reg.table {
		case tableHolding:
			for index, value := range reg.words {
				s.holding[reg.address+index] = value
			}
		case tableInput:
			for index, value := range reg.words {
				s.input[reg.address+index] = value
			}
		case tableCoils:
			for index, value := range reg.bits {
				s.coils[reg.address+index] = value
			}
		case tableDiscrete:
			for index, value := range reg.bits {
				s.discrete[reg.address+index] = value
			}
		}
	}
}

func (s *modbusServer) rangeConfigured(table registerTable, address, count int) bool {
	if address < 0 || count < 1 || address+count > 65536 {
		return false
	}
	for index := address; index < address+count; index++ {
		if s.registerAt(table, index) == nil {
			return false
		}
	}
	return true
}

func (s *modbusServer) registerAt(table registerTable, address int) *register {
	return s.index[table][address]
}

func (s *modbusServer) affectedRegisters(table registerTable, address, count int) []*register {
	unique := make(map[*register]struct{})
	for index := address; index < address+count; index++ {
		if reg := s.registerAt(table, index); reg != nil {
			unique[reg] = struct{}{}
		}
	}
	registers := make([]*register, 0, len(unique))
	for reg := range unique {
		registers = append(registers, reg)
	}
	sort.Slice(registers, func(i, j int) bool {
		return registers[i].address < registers[j].address
	})
	return registers
}

func (s *modbusServer) registerValue(reg *register) interface{} {
	switch reg.dataType {
	case typeWord:
		return s.wordAt(reg, 0)
	case typeFloat:
		bits := uint32(s.wordAt(reg, 0))<<16 | uint32(s.wordAt(reg, 1))
		return math.Float32frombits(bits)
	case typeString:
		bytes := make([]byte, reg.length*2)
		for index := 0; index < reg.length; index++ {
			binary.BigEndian.PutUint16(bytes[index*2:], s.wordAt(reg, index))
		}
		return strings.TrimRight(string(bytes), "\x00")
	case typeBool:
		return s.bitAt(reg, 0)
	default:
		return nil
	}
}

func (s *modbusServer) wordAt(reg *register, offset int) uint16 {
	if reg.table == tableInput {
		return s.input[reg.address+offset]
	}
	return s.holding[reg.address+offset]
}

func (s *modbusServer) bitAt(reg *register, offset int) bool {
	if reg.table == tableDiscrete {
		return s.discrete[reg.address+offset]
	}
	return s.coils[reg.address+offset]
}

type registerTable int

const (
	tableHolding registerTable = iota
	tableInput
	tableCoils
	tableDiscrete
)

type registerType int

const (
	typeWord registerType = iota
	typeFloat
	typeString
	typeBool
)

type register struct {
	name     string
	address  int
	dataType registerType
	typeName string
	function string
	table    registerTable
	length   int
	words    []uint16
	bits     []bool
}

func (r *register) event(value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":        r.name,
		"address":     r.address,
		"type":        r.typeName,
		"mb-function": r.function,
		"data":        value,
	}
}

func validateSettings(settings *Settings) error {
	if settings.Port < 1 || settings.Port > 65535 {
		return fmt.Errorf("setting %q must be between 1 and 65535", "port")
	}
	if settings.SlaveID < 1 || settings.SlaveID > 247 {
		return fmt.Errorf("setting %q must be between 1 and 247", "slaveId")
	}
	if len(settings.Registers) == 0 {
		return fmt.Errorf("setting %q must contain at least one register", "registers")
	}
	return nil
}

func parseRegisters(raw []interface{}) ([]*register, error) {
	registers := make([]*register, 0, len(raw))
	occupied := make(map[registerTable]map[int]bool)
	for index, entry := range raw {
		fields, ok := entry.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("registers[%d] must be an object", index)
		}
		name, ok := fields["name"].(string)
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("registers[%d].name must be a non-empty string", index)
		}
		address, err := coerce.ToInt(fields["address"])
		if err != nil || address < 0 || address > 65535 {
			return nil, fmt.Errorf("register %q address must be an integer between 0 and 65535", name)
		}
		typeName, ok := fields["type"].(string)
		if !ok {
			return nil, fmt.Errorf("register %q type must be a string", name)
		}
		dataType, normalizedType, err := parseRegisterType(typeName)
		if err != nil {
			return nil, fmt.Errorf("register %q: %w", name, err)
		}
		function, ok := fields["mb-function"].(string)
		if !ok {
			return nil, fmt.Errorf("register %q mb-function must be a string", name)
		}
		table, function, err := parseRegisterFunction(function)
		if err != nil {
			return nil, fmt.Errorf("register %q: %w", name, err)
		}
		if dataType == typeBool && table != tableCoils && table != tableDiscrete {
			return nil, fmt.Errorf("register %q: Bool requires a coil or discrete-input function", name)
		}
		if dataType != typeBool && (table == tableCoils || table == tableDiscrete) {
			return nil, fmt.Errorf("register %q: type %s requires a 16-bit register function", name, normalizedType)
		}
		data, exists := fields["data"]
		if !exists || data == nil {
			return nil, fmt.Errorf("register %q data is required", name)
		}

		reg := &register{
			name:     name,
			address:  address,
			dataType: dataType,
			typeName: typeName,
			function: function,
			table:    table,
		}
		if err := initializeRegister(reg, data); err != nil {
			return nil, fmt.Errorf("register %q: %w", name, err)
		}
		if address+reg.length > 65536 {
			return nil, fmt.Errorf("register %q exceeds the Modbus address range", name)
		}
		if occupied[table] == nil {
			occupied[table] = make(map[int]bool)
		}
		for cell := address; cell < address+reg.length; cell++ {
			if occupied[table][cell] {
				return nil, fmt.Errorf("register %q overlaps another register at address %d", name, cell)
			}
			occupied[table][cell] = true
		}
		registers = append(registers, reg)
	}
	return registers, nil
}

func initializeRegister(reg *register, data interface{}) error {
	switch reg.dataType {
	case typeWord:
		value, err := coerce.ToInt(data)
		if err != nil {
			return fmt.Errorf("Word data must be an integer: %w", err)
		}
		if value < 0 || value > 65535 {
			return fmt.Errorf("Word data must be between 0 and 65535")
		}
		reg.length = 1
		reg.words = []uint16{uint16(value)}
	case typeFloat:
		value, err := coerce.ToFloat32(data)
		if err != nil {
			return fmt.Errorf("Float data must be a number: %w", err)
		}
		bits := math.Float32bits(value)
		reg.length = 2
		reg.words = []uint16{uint16(bits >> 16), uint16(bits)}
	case typeString:
		value, ok := data.(string)
		if !ok {
			return fmt.Errorf("String data must be a string")
		}
		reg.length = (len(value) + 1) / 2
		if reg.length == 0 {
			reg.length = 1
		}
		bytes := make([]byte, reg.length*2)
		copy(bytes, []byte(value))
		reg.words = make([]uint16, reg.length)
		for index := range reg.words {
			reg.words[index] = binary.BigEndian.Uint16(bytes[index*2:])
		}
	case typeBool:
		value, err := coerce.ToBool(data)
		if err != nil {
			return fmt.Errorf("Bool data must be boolean: %w", err)
		}
		reg.length = 1
		reg.bits = []bool{value}
	}
	return nil
}

func parseRegisterType(value string) (registerType, string, error) {
	switch strings.ToLower(value) {
	case "word":
		return typeWord, "Word", nil
	case "float", "float32":
		return typeFloat, "Float", nil
	case "string":
		return typeString, "String", nil
	case "bool", "boolean":
		return typeBool, "Bool", nil
	default:
		return 0, "", fmt.Errorf("unsupported type %q", value)
	}
}

func parseRegisterFunction(value string) (registerTable, string, error) {
	switch strings.ToLower(value) {
	case "readcoils", "writesinglecoil", "writemultiplecoils":
		return tableCoils, value, nil
	case "readdiscreteinputs":
		return tableDiscrete, value, nil
	case "readholdingregisters", "writesingleregister", "writemultipleregisters":
		return tableHolding, value, nil
	case "readinputregisters":
		return tableInput, value, nil
	default:
		return 0, "", fmt.Errorf("unsupported mb-function %q", value)
	}
}

func indexRegisters(registers []*register) map[registerTable]map[int]*register {
	index := make(map[registerTable]map[int]*register)
	for _, reg := range registers {
		if index[reg.table] == nil {
			index[reg.table] = make(map[int]*register, reg.length)
		}
		for address := reg.address; address < reg.address+reg.length; address++ {
			index[reg.table][address] = reg
		}
	}
	return index
}

func tableForFunction(function int) registerTable {
	switch function {
	case functionReadCoils:
		return tableCoils
	case functionReadDiscreteInputs:
		return tableDiscrete
	case functionReadInputRegisters:
		return tableInput
	default:
		return tableHolding
	}
}

func exceptionResponse(function, code int) []byte {
	return []byte{byte(function) | 0x80, byte(code)}
}

func isClosedNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	if opErr, ok := err.(*net.OpError); ok && opErr.Err != nil && opErr.Err.Error() == "use of closed network connection" {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}
