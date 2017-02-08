// Package mojo is a client to the Mojo API
package mojo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// ErrDuplicate is returned by AddContact when a contact with same id
// already exists in Mojo
var ErrDuplicate = errors.New("mojo: contact with same ID already exists")

// Mojo client
type Mojo struct {
	// URL for this account, including protocol + host, example:
	// https://posttest.mojosells.com
	//
	// Each mojo client has their own url
	URL string

	// Token is the access token provided by Mojo after the client
	// has logged in using OAuth
	Token string

	HTTP *http.Client // (optional) http client to perform requests
}

// AddContact creates a new Contact in Mojo
//
// Contact ID and GroupID must be provided. At least one contact field should be provided
//
// Return ErrDuplicate if a contact with same ID already exists. Return other errors
// if can't make the request of if Mojo returns an error
func (mj *Mojo) AddContact(c Contact) error {
	reqbody, err := json.Marshal([]Contact{c})
	if err != nil {
		return fmt.Errorf("mojo: encoding body (%v)", err)
	}
	req, err := http.NewRequest("POST", mj.URL+"/api/contacts/bulk_create/", bytes.NewReader(reqbody))
	if err != nil {
		return fmt.Errorf("mojo: building request (%v)", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mj.Token)
	if mj.HTTP == nil {
		mj.HTTP = &http.Client{Timeout: 3 * time.Second}
	}
	res, err := mj.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("mojo: making request (%v)", err)
	}
	defer res.Body.Close()
	resbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("mojo: reading response (%v)", err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("mojo: invalid status code %d with body %v", res.StatusCode, string(resbody))
	}
	var data mojoResponse
	if err := json.Unmarshal(resbody, &data); err != nil {
		return fmt.Errorf("mojo: decoding response body (%v)", err)
	}
	if len(data.Errors) == 1 && strings.HasPrefix(data.Errors[0], "Duplicated 'api_contact_id':") {
		return ErrDuplicate
	}
	return nil
}

type mojoResponse struct {
	Errors []string `json:"errors"`
}

// Contact to be created in Mojo
//
// Either Name must provided OR at least one of Email, MobilePhone, WorkPhone, HomePhone
type Contact struct {
	ID          string // required
	GroupID     int    // required
	Name        string
	Address     string
	Email       string
	MobilePhone string
	WorkPhone   string
	HomePhone   string
}

func (c Contact) MarshalJSON() ([]byte, error) {
	if c.ID == "" {
		return []byte{}, errors.New("missing required field ID")
	}
	if c.GroupID == 0 {
		return []byte{}, errors.New("missing required field GroupID")
	}
	cc := contact{
		ID:      c.ID,
		Name:    c.Name,
		Address: c.Address,
		Group:   []map[string]int{{"group_id": c.GroupID}},
	}
	if c.WorkPhone != "" {
		cc.Media = append(cc.Media, media{1, cleanPhone(c.WorkPhone)})
	}
	if c.MobilePhone != "" {
		cc.Media = append(cc.Media, media{2, cleanPhone(c.MobilePhone)})
	}
	if c.HomePhone != "" {
		cc.Media = append(cc.Media, media{3, cleanPhone(c.HomePhone)})
	}
	if c.Email != "" {
		cc.Media = append(cc.Media, media{4, c.Email})
	}
	return json.Marshal(cc)
}

func cleanPhone(ph string) string {
	ph = strings.Replace(ph, "(", "", -1)
	ph = strings.Replace(ph, ")", "", -1)
	ph = strings.Replace(ph, "-", "", -1)
	ph = strings.Replace(ph, " ", "", -1)
	return ph
}

type contact struct {
	ID      string           `json:"api_contact_id"`
	Name    string           `json:"full_name"`
	Group   []map[string]int `json:"contactgroup_set"`
	Address string           `json:"address,omitempty"`
	// list of phones and emails
	// 1-work, 2-mobile, 3-home, 4-email, 5-other
	Media []media `json:"mediainfo_set,omitempty"`
}

type media struct {
	Type  int    `json:"type"`
	Value string `json:"value"`
}
