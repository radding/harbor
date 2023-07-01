package plugins

import "github.com/radding/harbor-plugins/proto"

type RegisterCommandRequest proto.RegisterCommandRequest
type RegisterCommandResponse proto.RegisterCommandResponse

type Command interface {
	RegisterCommand(RegisterCommandRequest) (RegisterCommandResponse, error)
}
