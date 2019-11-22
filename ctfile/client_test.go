package client

import (
	"encoding/json"
	"testing"
)

func TestClient_GetShareInfo(t *testing.T) {
	c := NewClient()
	share, err := c.GetShareInfo("11449240-32213899-2b9439", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(share)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}

var pubCookie = "B2NWZVRrVzZQZABkA2EHaQFaVGxRDVJmVDRQMQM4BzZRN1RnBTgGY1NgVjNRNlY_AWNdaQQRBTUHOlBiC2gGagd_VjRUN1dvUFwAFgMNB3YBMFQsUWBSQFRdUHEDSAdtUVZUMQU-BgxTMFZTUWhWTQElXVQEMgViB2VQMgswBjYHY1ZmVGhXNFBkAGQDZgdsAWdUFFFjUmhUZVBqA2UHKVFkVDoFZAZaU2dWN1ExVjkBYV1vBGYFYwdiUDQLCAY2B2FWY1RqV2BQNgAxA20HOAFmVDVRZVI2VD1QMgM-B2NRNFRhBWoGY1NnVjNRMlY-ATBdPgRpBWEHZ1BnC2c"

func TestClient_GetDownloadUrl(t *testing.T) {
	c := NewClient()
	if err := c.SetCookies(pubCookie); err != nil {
		t.Fatal(err)
	}
	t.Log(c.GetDownloadUrl(&File{
		ID: "tempdir-UDBTZV1rCmZUb1M2AjcEZgYpV2VQYVhpCGdWNQBnUG8BYlVnVHsBaFZhAmIFMAFhVmcFN1FiDTkMYg",
	}))

}
