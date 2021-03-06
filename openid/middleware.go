package openid

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
)

// The Configuration contains the entities needed to perform ID token validation.
// This type should be instantiated at the application startup time.
type Configuration struct {
	tokenValidator jwtTokenValidator
	idTokenGetter  GetIDTokenFunc
	errorHandler   ErrorHandlerFunc
}

type option func(*Configuration) error

// NewConfiguration creates a new instance of Configuration and returns a pointer to it.
// This function receives a collection of the function type option. Each of those functions are
// responsible for setting some part of the returned *Configuration. If any if the option functions
// returns an error then NewConfiguration will return a nil configuration and that error.
func NewConfiguration(options ...option) (*Configuration, error) {
	m := new(Configuration)
	cp := newHTTPConfigurationProvider(defaultHTTPGet, &jsonConfigurationDecoder{})
	jp := newHTTPJwksProvider(defaultHTTPGet, &jsonJwksDecoder{})
	ksp := newSigningKeySetProvider(cp, jp, &pemPublicKeyEncoder{})
	kp := newSigningKeyProvider(ksp)
	m.tokenValidator = newIDTokenValidator(nil, jwtParserFunc(jwt.Parse), kp, &defaultPemToRSAPublicKeyParser{})

	for _, option := range options {
		err := option(m)

		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

// ProvidersGetter option registers the function responsible for returning the
// providers containing the valid issuer and client IDs used to validate the ID Token.
func ProvidersGetter(pg GetProvidersFunc) func(*Configuration) error {
	return func(c *Configuration) error {
		c.tokenValidator.(*idTokenValidator).provGetter = pg
		return nil
	}
}

// ErrorHandler option registers the function responsible for handling
// the errors returned during token validation. When this option is not used then the
// middleware will use the default internal implementation validationErrorToHTTPStatus.
func ErrorHandler(eh ErrorHandlerFunc) func(*Configuration) error {
	return func(c *Configuration) error {
		c.errorHandler = eh
		return nil
	}
}

// HTTPGetFunc is a function that gets a URL based on a contextual request
// and a target URL. The default behavior is the http.Get method, ignoring
// the request parameter.
type HTTPGetFunc func(r *http.Request, url string) (*http.Response, error)

var defaultHTTPGet = func(r *http.Request, url string) (*http.Response, error) {
	return http.Get(url)
}

// HTTPGetter option registers the function responsible for returning the
// providers containing the valid issuer and client IDs used to validate the ID Token.
func HTTPGetter(hg HTTPGetFunc) func(*Configuration) error {
	return func(c *Configuration) error {
		sksp := c.tokenValidator.(*idTokenValidator).
			keyGetter.(*signingKeyProvider).
			keySetGetter.(*signingKeySetProvider)
		sksp.configGetter.(*httpConfigurationProvider).getter = hg
		sksp.jwksGetter.(*httpJwksProvider).getter = hg
		return nil
	}
}

// Authenticate middleware performs the validation of the OIDC ID Token.
// If an error happens, i.e.: expired token, the next handler may or may not executed depending on the
// provided ErrorHandlerFunc option. The default behavior, determined by validationErrorToHTTPStatus,
// stops the execution and returns Unauthorized.
// If the validation is successful then the next handler(h) will be executed.
func Authenticate(conf *Configuration, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, halt := authenticate(conf, w, r); !halt {
			h.ServeHTTP(w, r)
		}
	})
}

// Authenticate middleware performs the validation of the OIDC ID Token.
// If an error happens, i.e.: expired token, the next handler may or may not execute depending on the
// provided ErrorHandlerFunc option. The default behavior, determined by validationErrorToHTTPStatus,
// stops the execution and returns Unauthorized.
// If the validation is successful then the next handler(h) will be executed.
func AuthenticateWithParams(conf *Configuration, h httprouter.Handle) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if _, halt := authenticate(conf, w, r); !halt {
			h(w, r, params)
		}
	})
}

// AuthenticateUser middleware performs the validation of the OIDC ID Token and
// forwards the authenticated user's information to the next handler in the pipeline.
// If an error happens, i.e.: expired token, the next handler may or may not executed depending on the
// provided ErrorHandlerFunc option. The default behavior, determined by validationErrorToHTTPStatus,
// stops the execution and returns Unauthorized.
// If the validation is successful then the next handler(h) will be executed and will
// receive the authenticated user information.
func AuthenticateUser(conf *Configuration, h UserHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, halt := authenticateUser(conf, w, r); !halt {
			h(u, w, r)
		}
	})
}

// AuthenticateUser middleware performs the validation of the OIDC ID Token and
// forwards the authenticated user's information to the next handler in the pipeline.
// If an error happens, i.e.: expired token, the next handler may or may not executed depending on the
// provided ErrorHandlerFunc option. The default behavior, determined by validationErrorToHTTPStatus,
// stops the execution and returns Unauthorized.
// If the validation is successful then the next handler(h) will be executed and will
// receive the authenticated user information.
func AuthenticateUserWithParams(conf *Configuration, h UserHandlerWithParams) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if u, halt := authenticateUser(conf, w, r); !halt {
			h(u, w, r, params)
		}
	})
}

func authenticate(c *Configuration, rw http.ResponseWriter, req *http.Request) (t *jwt.Token, halt bool) {
	var tg GetIDTokenFunc
	if c.idTokenGetter == nil {
		tg = getIDTokenAuthorizationHeader
	} else {
		tg = c.idTokenGetter
	}

	var eh ErrorHandlerFunc
	if c.errorHandler == nil {
		eh = validationErrorToHTTPStatus
	} else {
		eh = c.errorHandler
	}

	ts, err := tg(req)

	if err != nil {
		return nil, eh(err, rw, req)
	}

	vt, err := c.tokenValidator.validate(req, ts)

	if err != nil {
		return nil, eh(err, rw, req)
	}

	return vt, false
}

func authenticateUser(c *Configuration, rw http.ResponseWriter, req *http.Request) (u *User, halt bool) {
	var vt *jwt.Token

	var eh ErrorHandlerFunc
	if c.errorHandler == nil {
		eh = validationErrorToHTTPStatus
	} else {
		eh = c.errorHandler
	}

	if t, halt := authenticate(c, rw, req); !halt {
		vt = t
	} else {
		return nil, halt
	}

	u, err := newUser(vt)

	if err != nil {
		return nil, eh(err, rw, req)
	}

	return u, false
}
