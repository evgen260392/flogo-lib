package opcuaclient

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

type stubReadClient struct {
	response *ua.ReadResponse
	err      error
	request  *ua.ReadRequest
}

func (c *stubReadClient) Read(request *ua.ReadRequest) (*ua.ReadResponse, error) {
	c.request = request
	return c.response, c.err
}

func TestNewRequiresNodeID(t *testing.T) {
	settings := &Settings{Endpoint: "opc.tcp://localhost:4840"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)

	_, err := New(iCtx)
	assert.Error(t, err)
}

func TestNewValidatesNodeID(t *testing.T) {
	settings := &Settings{
		Endpoint: "opc.tcp://localhost:4840",
		NodeID:   "not-a-node-id",
	}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)

	_, err := New(iCtx)
	assert.Error(t, err)
}

func TestReadNodeReturnsValue(t *testing.T) {
	nodeID, err := ua.ParseNodeID("ns=2;s=Temperature")
	if err != nil {
		t.Fatalf("parsing node ID: %v", err)
	}
	client := &stubReadClient{
		response: &ua.ReadResponse{
			Results: []*ua.DataValue{{
				Status: ua.StatusOK,
				Value:  ua.MustVariant(float64(21.5)),
			}},
		},
	}

	value, err := readNode(client, nodeID)
	if err != nil {
		t.Fatalf("reading node: %v", err)
	}
	assert.Equal(t, 21.5, value)
	assert.Equal(t, ua.AttributeIDValue, client.request.NodesToRead[0].AttributeID)
	assert.Equal(t, "ns=2;s=Temperature", client.request.NodesToRead[0].NodeID.String())
}

func TestReadNodeAcceptsGoodStatusWithInfoBits(t *testing.T) {
	nodeID, err := ua.ParseNodeID("ns=2;i=42")
	if err != nil {
		t.Fatalf("parsing node ID: %v", err)
	}
	client := &stubReadClient{
		response: &ua.ReadResponse{
			Results: []*ua.DataValue{{
				Status: ua.StatusGoodClamped,
				Value:  ua.MustVariant(int32(42)),
			}},
		},
	}

	value, err := readNode(client, nodeID)
	if err != nil {
		t.Fatalf("reading node: %v", err)
	}
	assert.Equal(t, int32(42), value)
}

func TestReadNodeReturnsServerStatusError(t *testing.T) {
	nodeID, err := ua.ParseNodeID("ns=2;i=42")
	if err != nil {
		t.Fatalf("parsing node ID: %v", err)
	}
	client := &stubReadClient{
		response: &ua.ReadResponse{
			Results: []*ua.DataValue{{Status: ua.StatusBadNodeIDUnknown}},
		},
	}

	_, err = readNode(client, nodeID)
	assert.Error(t, err)
}
