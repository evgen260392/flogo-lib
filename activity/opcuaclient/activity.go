package opcuaclient

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
)

const ovValue = "value"

type Settings struct {
	Endpoint          string `md:"endpoint,required"`
	NodeID            string `md:"nodeID,required"`
	SecurityPolicy    string `md:"securityPolicy"`
	SecurityMode      string `md:"securityMode,allowed(None,Sign,SignAndEncrypt)"`
	Username          string `md:"username"`
	Password          string `md:"password"`
	UserTokenPolicyID string `md:"userTokenPolicyID"`
	Timeout           int    `md:"timeout"`
}

type Output struct {
	Value interface{} `md:"value"`
}

func init() {
	_ = activity.Register(&Activity{}, New)
}

var activityMd = activity.ToMetadata(&Settings{}, &Output{})

type Activity struct {
	settings *Settings
	nodeID   *ua.NodeID
}

func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{
		SecurityPolicy: "None",
		SecurityMode:   "None",
		Timeout:        10,
	}
	if err := metadata.MapToStruct(ctx.Settings(), settings, true); err != nil {
		return nil, err
	}

	nodeID, err := validateSettings(settings)
	if err != nil {
		return nil, err
	}

	return &Activity{settings: settings, nodeID: nodeID}, nil
}

func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

func (a *Activity) Eval(ctx activity.Context) (bool, error) {
	settings := a.settings
	timeout := time.Duration(settings.Timeout) * time.Second

	options := []opcua.Option{
		opcua.SecurityPolicy(settings.SecurityPolicy),
		opcua.SecurityModeString(settings.SecurityMode),
		opcua.RequestTimeout(timeout),
	}
	if settings.Username != "" {
		options = append(options, opcua.AuthUsername(settings.Username, settings.Password))
		if settings.UserTokenPolicyID != "" {
			options = append(options, opcua.AuthPolicyID(settings.UserTokenPolicyID))
		}
	}

	client := opcua.NewClient(settings.Endpoint, options...)
	connectCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(connectCtx); err != nil {
		return false, fmt.Errorf("connecting to OPC UA endpoint %q: %v", settings.Endpoint, err)
	}
	defer client.Close()

	value, err := readNode(client, a.nodeID)
	if err != nil {
		return false, err
	}
	if err := ctx.SetOutput(ovValue, value); err != nil {
		return false, err
	}

	return true, nil
}

type readClient interface {
	Read(*ua.ReadRequest) (*ua.ReadResponse, error)
}

func readNode(client readClient, nodeID *ua.NodeID) (interface{}, error) {
	response, err := client.Read(&ua.ReadRequest{
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead: []*ua.ReadValueID{{
			NodeID:      nodeID,
			AttributeID: ua.AttributeIDValue,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("reading OPC UA node %q: %v", nodeID, err)
	}
	if response == nil || len(response.Results) == 0 || response.Results[0] == nil {
		return nil, fmt.Errorf("OPC UA server returned no value for node %q", nodeID)
	}
	result := response.Results[0]
	if uint32(result.Status)&0xC0000000 != 0 {
		return nil, fmt.Errorf("reading OPC UA node %q returned status %v", nodeID, result.Status)
	}
	if result.Value == nil {
		return nil, fmt.Errorf("OPC UA server returned an empty value for node %q", nodeID)
	}

	return result.Value.Value(), nil
}

func validateSettings(settings *Settings) (*ua.NodeID, error) {
	endpoint, err := url.Parse(settings.Endpoint)
	if err != nil || endpoint.Scheme != "opc.tcp" || endpoint.Host == "" {
		return nil, fmt.Errorf("setting %q must be a valid opc.tcp endpoint", "endpoint")
	}
	if settings.NodeID == "" {
		return nil, fmt.Errorf("setting %q is required", "nodeID")
	}

	nodeID, err := ua.ParseNodeID(settings.NodeID)
	if err != nil {
		return nil, fmt.Errorf("setting %q is invalid: %v", "nodeID", err)
	}
	if settings.Timeout <= 0 {
		return nil, fmt.Errorf("setting %q must be greater than zero", "timeout")
	}
	if settings.SecurityMode != "None" && settings.SecurityMode != "Sign" && settings.SecurityMode != "SignAndEncrypt" {
		return nil, fmt.Errorf("setting %q must be None, Sign, or SignAndEncrypt", "securityMode")
	}
	if settings.Username == "" && (settings.Password != "" || settings.UserTokenPolicyID != "") {
		return nil, fmt.Errorf("setting %q requires username", "password/userTokenPolicyID")
	}

	return nodeID, nil
}
