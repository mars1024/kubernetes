package serviceaccount

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"gopkg.in/square/go-jose.v2/jwt"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
)

func TenantWiseJWTTokenAuthenticator(iss string, keys []interface{}) *JWTTokenAuthenticator {
	return &JWTTokenAuthenticator{
		iss:       iss,
		keys:      keys,
		validator: &tenantWiseLegacyValidator{},
	}
}

type JWTTokenAuthenticator struct {
	iss       string
	keys      []interface{}
	validator *tenantWiseLegacyValidator
}

func (j *JWTTokenAuthenticator) AuthenticateToken(tokenData string) (user.Info, bool, error) {
	if !j.hasCorrectIssuer(tokenData) {
		return nil, false, nil
	}

	tok, err := jwt.ParseSigned(tokenData)
	if err != nil {
		return nil, false, nil
	}

	public := &jwt.Claims{}
	private := j.validator.NewPrivateClaims()

	var (
		found   bool
		errlist []error
	)
	for _, key := range j.keys {
		if err := tok.Claims(key, public, private); err != nil {
			errlist = append(errlist, err)
			continue
		}
		found = true
		break
	}

	if !found {
		return nil, false, utilerrors.NewAggregate(errlist)
	}

	// If we get here, we have a token with a recognized signature and
	// issuer string.
	tenant, ns, name, uid, err := j.validator.Validate(tokenData, public, private)
	if err != nil {
		return nil, false, err
	}

	user := &user.DefaultInfo{
		Name:   apiserverserviceaccount.MakeUsername(ns, name),
		UID:    uid,
		Groups: apiserverserviceaccount.MakeGroupNames(ns),
		Extra:  make(map[string][]string),
	}

	if err := util.TransformTenantInfoToUser(tenant, user); err != nil {
		return nil, false, err
	}
	return user, true, nil
}

func (j *JWTTokenAuthenticator) hasCorrectIssuer(tokenData string) bool {
	parts := strings.Split(tokenData, ".")
	if len(parts) != 3 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	claims := struct {
		// WARNING: this JWT is not verified. Do not trust these claims.
		Issuer string `json:"iss"`
	}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	if claims.Issuer != j.iss {
		return false
	}
	return true

}
