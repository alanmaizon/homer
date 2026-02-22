package api

import "github.com/alanmaizon/homer/backend/internal/connectors"

var newConnectorFromEnv = connectors.NewConnectorFromEnv
var newGoogleDocsOAuthManagerFromEnv = connectors.NewGoogleDocsOAuthManagerFromEnv
