package security

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Sub    string   `json:"sub"`
	Scopes []string `json:"scopes"`
	JTI    string   `json:"jti"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
	issuer     string
	audience   string
	ttl        time.Duration
}

func NewTokenManager(privateKeyPath, publicKeyPath, keyID, issuer, audience string, ttl time.Duration) (*TokenManager, error) {
	priv, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}

	pub, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return nil, err
	}

	return &TokenManager{
		privateKey: priv,
		publicKey:  pub,
		keyID:      keyID,
		issuer:     issuer,
		audience:   audience,
		ttl:        ttl,
	}, nil
}

func (tm *TokenManager) Generate(subject string, scopes []string, jti string) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		Sub:    subject,
		Scopes: scopes,
		JTI:    jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Audience:  jwt.ClaimStrings{tm.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.ttl)),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tm.keyID

	return token.SignedString(tm.privateKey)
}

func (tm *TokenManager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return tm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token or claims")
	}

	// Future: check blacklist for revoked tokens using claims.JTI
	// if revoked(claims.JTI) { return nil, errors.New("token revoked") }

	return claims, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid private key format")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid public key format")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pub, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return pub, nil
}

func RequireScope(claims *Claims, required string) bool {
	for _, scope := range claims.Scopes {
		if scope == required {
			return true
		}
	}
	return false
}
