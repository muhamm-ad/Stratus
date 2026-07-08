package core

// CloudProviderConstraint is an OPTIONAL interface a CloudProvider may implement to
// restrict which identity issuers it can federate from. Azure implements it
// because Azure Resource Manager only accepts Microsoft Entra ID tokens: a token
// from another OIDC issuer (Okta, Keycloak, …) cannot obtain ARM access. Other
// cloud providers don't implement it and accept any issuer.
type CloudProviderConstraint interface {
	// AcceptsIdentityIssuer reports whether the cloud provider can federate from an identity 
	AcceptsIdentityIssuer(IdentityIssuer string) bool
}

// issuerer is implemented by identity providers that can report their OIDC
// issuer, letting the service evaluate an IdentityConstraint.
type issuerer interface {
	Issuer() string
}

// IssuerOf returns an identity provider's OIDC issuer, or "" if it can't report one.
func IssuerOf(idp IdentityProvider) string {
	if i, ok := idp.(issuerer); ok {
		return i.Issuer()
	}
	return ""
}
