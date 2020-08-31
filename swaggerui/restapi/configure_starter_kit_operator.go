// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/yamlpc"

	"github.com/ibm/starter-kit-operator/swaggerui/restapi/operations"
	"github.com/ibm/starter-kit-operator/swaggerui/restapi/operations/core_v1"
	"github.com/ibm/starter-kit-operator/swaggerui/restapi/operations/starter_kit_operations"
)

//go:generate swagger generate server --target ../../swaggerui --name StarterKitOperator --spec ../swagger-template.yaml --principal interface{}

func configureFlags(api *operations.StarterKitOperatorAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.StarterKitOperatorAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.YamlConsumer = yamlpc.YAMLConsumer()

	api.JSONProducer = runtime.JSONProducer()
	api.ProtobufProducer = runtime.ProducerFunc(func(w io.Writer, data interface{}) error {
		return errors.NotImplemented("protobuf producer has not yet been implemented")
	})
	api.YamlProducer = yamlpc.YAMLProducer()

	// Applies when the "Authorization" header is set
	if api.BearerAuth == nil {
		api.BearerAuth = func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (Bearer) Authorization from header param [Authorization] has not yet been implemented")
		}
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()
	if api.CoreV1GetAPIV1SecretsLabelSelectorDevxHandler == nil {
		api.CoreV1GetAPIV1SecretsLabelSelectorDevxHandler = core_v1.GetAPIV1SecretsLabelSelectorDevxHandlerFunc(func(params core_v1.GetAPIV1SecretsLabelSelectorDevxParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation core_v1.GetAPIV1SecretsLabelSelectorDevx has not yet been implemented")
		})
	}
	if api.StarterKitOperationsGetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsHandler == nil {
		api.StarterKitOperationsGetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsHandler = starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsHandlerFunc(func(params starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkits has not yet been implemented")
		})
	}
	if api.StarterKitOperationsGetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameHandler == nil {
		api.StarterKitOperationsGetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameHandler = starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameHandlerFunc(func(params starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsName has not yet been implemented")
		})
	}
	if api.StarterKitOperationsGetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameStatusHandler == nil {
		api.StarterKitOperationsGetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameStatusHandler = starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameStatusHandlerFunc(func(params starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameStatusParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation starter_kit_operations.GetApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameStatus has not yet been implemented")
		})
	}
	if api.StarterKitOperationsGetApisDevxIbmComV1alpha1StarterkitsHandler == nil {
		api.StarterKitOperationsGetApisDevxIbmComV1alpha1StarterkitsHandler = starter_kit_operations.GetApisDevxIbmComV1alpha1StarterkitsHandlerFunc(func(params starter_kit_operations.GetApisDevxIbmComV1alpha1StarterkitsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation starter_kit_operations.GetApisDevxIbmComV1alpha1Starterkits has not yet been implemented")
		})
	}
	if api.StarterKitOperationsPostApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsHandler == nil {
		api.StarterKitOperationsPostApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsHandler = starter_kit_operations.PostApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsHandlerFunc(func(params starter_kit_operations.PostApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation starter_kit_operations.PostApisDevxIbmComV1alpha1NamespacesNamespaceStarterkits has not yet been implemented")
		})
	}
	if api.StarterKitOperationsPutApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameHandler == nil {
		api.StarterKitOperationsPutApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameHandler = starter_kit_operations.PutApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameHandlerFunc(func(params starter_kit_operations.PutApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsNameParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation starter_kit_operations.PutApisDevxIbmComV1alpha1NamespacesNamespaceStarterkitsName has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
