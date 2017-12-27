package httprestful

import (
	. "Elastos.ELA/net/httprestful/restful"
)

func StartServer() {
	rest := InitRestServer()
	rest.Start()
}
