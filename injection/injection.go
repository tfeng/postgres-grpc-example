package injection

import (
	"fmt"
	"github.com/tfeng/postgres-grpc-example/auth"
	"github.com/tfeng/postgres-grpc-example/models/user"
)

var (
	GrantTypeHandlers = grantTypeHandlers
)

var grantTypeHandlers = map[string]auth.GrantTypeHandler{
	auth.GrantType_client_credentials.String(): &auth.ClientCredentialsGrantTypeHandler{&clientStore{}},
	auth.GrantType_password.String():           &auth.UserPasswordGrantTypeHandler{&user.UserStore{}},
}

type clientStore struct{}

var clients = map[string]auth.ClientInfo{
	"client": {"password", []auth.Scope{auth.Scope_user_creation, auth.Scope_user_authorize}},
}

func (h *clientStore) GetClientInfo(clientId string) (*auth.ClientInfo, error) {
	clientInfo, ok := clients[clientId]
	if ok {
		return &clientInfo, nil
	} else {
		return nil, fmt.Errorf("Client not found")
	}
}
