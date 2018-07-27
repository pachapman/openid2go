package openid

import (
	"net/http"
	"github.com/julienschmidt/httprouter"
)

// The UserHandler represents a handler to be registered by the middleware AuthenticateUser.
// This handler allows the AuthenticateUser middleware to forward information about the the authenticated user to
// the rest of the application service.
//
// ServeHTTPWithUser is similar to the http.ServeHTTP function. It contains an additional paramater *User,
// which is used by the AuthenticateUser middleware to pass information about the authenticated user.
type UserHandler func(*User, http.ResponseWriter, *http.Request)


// The UserHandler represents a handler to be registered by the middleware AuthenticateUser.
// This handler allows the AuthenticateUser middleware to forward information about the the authenticated user to
// the rest of the application service.
//
// ServeHTTPWithUser is similar to the http.ServeHTTP function. It contains an additional paramater *User,
// which is used by the AuthenticateUser middleware to pass information about the authenticated user.
type UserHandlerWithParams func(*User, http.ResponseWriter, *http.Request, httprouter.Params)

//// The UserHandlerFunc is an adapter to allow the use of functions as UserHandler.
//// This is similar to using http.HandlerFunc as http.Handler
//type UserHandlerFunc func(*User, http.ResponseWriter, *http.Request)
//
//// ServeHTTPWithUser calls f(u, w, r)
//func (f UserHandlerFunc) ServeHTTPWithUser(u *User, w http.ResponseWriter, r *http.Request) {
//	f(u, w, r)
//}
