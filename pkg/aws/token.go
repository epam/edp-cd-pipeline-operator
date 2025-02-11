package aws

import (
	"fmt"

	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

type AIMAuthTokenGenerator interface {
	GetWithRole(clusterName, roleARN string) (Token, error)
}

type Token token.Token

type TokenGenerator struct {
	aws token.Generator
}

// GetWithRole returns a token for the given cluster and role ARN.
// The token is valid for 15 minutes.
func (t *TokenGenerator) GetWithRole(clusterName, roleARN string) (Token, error) {
	tkn, err := t.aws.GetWithRole(clusterName, roleARN)
	if err != nil {
		return Token{}, fmt.Errorf("failed to get token: %w", err)
	}

	return Token(tkn), nil
}

func NewTokenGenerator() (*TokenGenerator, error) {
	g, err := token.NewGenerator(false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create token generator: %w", err)
	}

	return &TokenGenerator{aws: g}, nil
}
