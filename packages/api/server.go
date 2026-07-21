package api

import "github.com/hariprakazz/cascade/packages/auth"

func Serve() string {
	return auth.Login()
}
