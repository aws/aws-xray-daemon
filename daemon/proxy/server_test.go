package proxy

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-xray-daemon/daemon/cfg"
	"github.com/stretchr/testify/assert"
)

// Assert that consume returns a ReadSeeker with the same content as the
// ReadCloser passed in.
func TestConsume(t *testing.T) {
	// Create an io.Reader
	r := strings.NewReader("Content")

	// Create an io.ReadCloser
	rc := ioutil.NopCloser(r)

	// Consume ReadCloser and create ReadSeeker
	rs, err := consume(rc)
	assert.Nil(t, err)

	// Read from ReadSeeker
	bytes, err := ioutil.ReadAll(rs)
	assert.Nil(t, err)

	// Assert contents of bytes are same as contents of original Reader
	assert.Equal(t, "Content", string(bytes))
}

// Assert that consume returns a nil ReadSeeker when a nil ReadCloser is passed in
func TestConsumeNilBody(t *testing.T) {
	// Create a nil io.ReadCloser
	var rc io.ReadCloser

	// Consume ReadCloser and create ReadSeeker
	rs, err := consume(rc)
	assert.Nil(t, err)
	assert.Nil(t, rs)
}

// Assert that Director modifies the passed in http.Request
func TestDirector(t *testing.T) {
	// Create dummy credentials to sign with
	cred := credentials.NewStaticCredentials("id", "secret", "token")

	// Create dummy aws Config
	awsCfg := &aws.Config{
		Endpoint:    aws.String("https://xray.us-east-1.amazonaws.com"),
		Region:      aws.String("us-east-1"),
		Credentials: cred,
	}

	// Create dummy aws Session
	sess := &session.Session{
		Config: awsCfg,
	}

	// Create proxy server
	s, err := NewServer(cfg.DefaultConfig(), awsCfg, sess)
	assert.Nil(t, err)

	// Extract director from server
	d := s.Handler.(*httputil.ReverseProxy).Director

	// Create http request to pass to director
	url, err := url.Parse("http://127.0.0.1:2000")
	assert.Nil(t, err)

	header := map[string][]string{
		"Connection": []string{},
	}

	req := &http.Request{
		URL:    url,
		Host:   "127.0.0.1",
		Header: header,
		Body:   ioutil.NopCloser(strings.NewReader("Body")),
	}

	// Apply director to request
	d(req)

	// Assert that the url was changed to point to AWS X-Ray
	assert.Equal(t, "https", req.URL.Scheme)
	assert.Equal(t, "xray.us-east-1.amazonaws.com", req.URL.Host)
	assert.Equal(t, "xray.us-east-1.amazonaws.com", req.Host)

	// Assert that additional headers were added by the signer
	assert.Contains(t, req.Header, "Authorization")
	assert.Contains(t, req.Header, "X-Amz-Security-Token")
	assert.Contains(t, req.Header, "X-Amz-Date")
	assert.NotContains(t, req.Header, "Connection")
}

// Fetching endpoint from aws config instance
func TestEndpoint1(t *testing.T) {
	e := "https://xray.us-east-1.amazonaws.com"
	awsCfg := &aws.Config{
		Endpoint: aws.String(e), // Endpoint value has higher priority than region value
		Region:   aws.String("us-west-1"),
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "Fetching endpoint from config instance")
	assert.Nil(t, err)
}

// Generating endpoint from region value of awsCfg instance
func TestEndpoint2(t *testing.T) {
	e := "https://xray.us-west-1.amazonaws.com"
	awsCfg := &aws.Config{
		Region: aws.String("us-west-1"), // No endpoint
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "Fetching endpoint from region")
	assert.Nil(t, err)
}

// Error received when no endpoint and region value present in awsCfg instance
func TestEndpoint3(t *testing.T) {
	awsCfg := &aws.Config{
	// No endpoint and region value
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, "", result, "Endpoint cannot be created")
	assert.NotNil(t, err)
}

func TestEndpoint4(t *testing.T) {
	awsCfg := &aws.Config{
		// region value set to ""
		Region: aws.String(""),
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, "", result, "Endpoint cannot be created")
	assert.NotNil(t, err)
}

func TestEndpoint5(t *testing.T) {
	e := "https://xray.us-west-1.amazonaws.com"
	awsCfg := &aws.Config{
		Endpoint: aws.String(""),          // Endpoint set to ""
		Region:   aws.String("us-west-1"), // No endpoint
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "Endpoint created from region value")
	assert.Nil(t, err)
}

// Testing AWS China partition
func TestEndpoint6(t *testing.T) {
	e := "https://xray.cn-northwest-1.amazonaws.com.cn"
	awsCfg := &aws.Config{
		Endpoint: aws.String(""),
		Region:   aws.String("cn-northwest-1"),
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "creating endpoint from region")
	assert.Nil(t, err)
}

// Testing AWS China partition
func TestEndpoint7(t *testing.T) {
	e := "https://xray.cn-north-1.amazonaws.com.cn"
	awsCfg := &aws.Config{
		Endpoint: aws.String(""),
		Region:   aws.String("cn-north-1"),
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "creating endpoint from region")
	assert.Nil(t, err)
}

// Testing AWS Gov partition
func TestEndpoint8(t *testing.T) {
	e := "https://xray.us-gov-east-1.amazonaws.com"
	awsCfg := &aws.Config{
		Endpoint: aws.String(""),
		Region:   aws.String("us-gov-east-1"),
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "creating endpoint from region")
	assert.Nil(t, err)
}

// Testing AWS Gov partition
func TestEndpoint9(t *testing.T) {
	e := "https://xray.us-gov-west-1.amazonaws.com"
	awsCfg := &aws.Config{
		Endpoint: aws.String(""),
		Region:   aws.String("us-gov-west-1"),
	}
	result, err := getServiceEndpoint(awsCfg)
	assert.Equal(t, e, result, "creating endpoint from region")
	assert.Nil(t, err)
}
