package spike_utils

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultRedisTimeout = 500 * time.Millisecond
)

type RedisInfo struct {
	Host     string
	Password string
	Timeout  time.Duration
}

func RedisGetInfo(redisUrlStr string, redisTimeout string) *RedisInfo {
	redisUrl, err := url.Parse(redisUrlStr)
	if err != nil {
		panic(err)
	}

	password, ok := redisUrl.User.Password()
	if !ok {
		password = ""
	}

	var timeout time.Duration
	timeoutInt, err := strconv.Atoi(redisTimeout)
	if err == nil {
		timeout = time.Duration(timeoutInt) * time.Second
	} else {
		timeout = DefaultRedisTimeout
	}

	return &RedisInfo{
		Host:     redisUrl.Host,
		Password: password,
		Timeout:  timeout,
	}
}

//Hash implements root.Hash
type Hash struct{}

// Dispose general
func Dispose() {
	if p := recover(); p != nil {
		stack := debug.Stack()
		log.Println(fmt.Sprintf("Panic found: %v\nStack: %v", p, string(stack)))
	}
}

//Generate a salted hash for the input string
func (c *Hash) Generate(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	hash := string(hashedBytes[:])
	return hash, nil
}

//Compare string to generated hash
func (c *Hash) Compare(hash string, s string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}

// GeneratePrivateKey string to crypto key private
func GeneratePrivateKey(hash string) (hashed string, err error) {
	if len(hash) == 0 {
		return "", errors.New("No input supplied")
	}

	hasher := sha256.New()
	hasher.Write([]byte(hash))

	stringToSHA256 := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	// Cut the length down to 32 bytes and return.
	return stringToSHA256[:32], nil
}

// GetBearer get
func GetBearer(r *http.Request) (accessToken string, ok bool) {
	auth := r.Header.Get("Authorization")
	prefix := "Bearer "

	if auth != "" && strings.HasPrefix(auth, prefix) {
		accessToken = auth[len(prefix):]
	} else {
		accessToken = r.FormValue("token")
	}

	if accessToken != "" {
		ok = true
	}

	return
}

func PointerFromInterface(data interface{}) interface{} {
	if reflect.ValueOf(data).Type().Kind() != reflect.Ptr {
		dataPtr := reflect.New(reflect.TypeOf(data))
		dataPtr.Elem().Set(reflect.ValueOf(data))
		data = dataPtr.Interface()
	}
	return data
}

func PatternFromEndpoint(patterns []rids.Pattern, specificEndpoint string) rids.Pattern {
	for _, pattern := range patterns {
		patternParts := strings.Split(pattern.EndpointName(), ".")
		queryParts := strings.Split(specificEndpoint, "?")
		endpointParts := strings.Split(queryParts[0], ".")

		if len(patternParts) != len(endpointParts) {
			continue
		}

		matches := true
		params := make(map[string]fmt.Stringer, 0)
		for i := range patternParts {
			if strings.HasPrefix(patternParts[i], "$") {
				key := patternParts[i][1:]
				params[key] = Stringer(endpointParts[i])
				continue
			}
			if patternParts[i] != endpointParts[i] {
				matches = false
				break
			}
		}

		if matches {
			var patternClone rids.Pattern
			patternClone = pattern.Clone()
			patternClone.SetParams(params)
			if len(queryParts) > 1 {
				patternClone.Query(queryParts[1])
			}
			return patternClone
		}
	}
	return nil
}
