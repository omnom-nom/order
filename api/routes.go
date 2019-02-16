package api

import (
	"fmt"
	"net/http"

	"github.com/omnom-nom/apiserver"
)

const (
	Apiv1 = "v1"
	ApiServiceType = "order"
)

var v1Prefix = fmt.Sprintf("%s/%s", Apiv1, ApiServiceType)
var routes = map[string][]apiserver.Route{
	v1Prefix: {
		{ Name: "HealthCheck",	Method: http.MethodGet,		Path: "healthcheck",		Handler: HealthCheck},
	//	{ Name: "CreateOrder",	Method: http.MethodPost,	Path: "create",			Handler: CreateOrder},
	//	{ Name: "OrderStatus",	Method: http.MethodGet,		Path: "status/{orderId}",	Handler: OrderStatus},
	//	{ Name: "DeleteOrder",	Method: http.MethodDelete,	Path: "delete/{orderId}",	Handler: DeleteOrder},
	},
}
