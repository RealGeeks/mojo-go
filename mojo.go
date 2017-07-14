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
type ErrDuplicate struct {
	IDs []string
}

func (e *ErrDuplicate) Error() string {
	return fmt.Sprintf("mojo: contacts already exist %s", strings.Join(e.IDs, ","))
}

// ErrInvalid is returned by AddContact when a validation error is detected,
// like missing a required field
type ErrInvalid struct {
	Msg string
}

func (e *ErrInvalid) Error() string {
	return fmt.Sprintf("mojo: %v", e.Msg)
}

// ErrForbidden is returned on status code 403, usually due to invalid
// access token
type ErrForbidden struct {
	Msg string
}

func (e *ErrForbidden) Error() string {
	return fmt.Sprintf("mojo: %v", e.Msg)
}

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

// AddNote adds a note to an existing contact
//
// Return ErrInvalid on validation errors and ErrForbidden if token is invalid
func (mj *Mojo) AddNote(contactID string, note string) error {
	data := map[string]interface{}{"api_contact_id": contactID, "contents": note, "type": 1}
	reqbody, err := json.Marshal(data)
	if err != nil {
		return &ErrInvalid{Msg: err.Error()}
	}
	url := prefixHTTP(mj.URL) + "/api/notes/"
	resbody, err := mj.post(url, reqbody)
	if err != nil {
		return err
	}
	var nfErr nonFieldErr
	if err := json.Unmarshal(resbody, &nfErr); err != nil {
		return fmt.Errorf("mojo: POST %s %s decoding %s (%v)", url, string(reqbody), string(reqbody), err)
	}
	if msg := nfErr.all(); msg != "" {
		return &ErrInvalid{Msg: fmt.Sprintf("POST %s %s validation error: %s", url, string(reqbody), nfErr.all())}
	}
	return nil
}

// possible error response body from AddNote that has status 200
// with body:
//
// {"non_field_errors": ["Invalid api_contact_id."]}
type nonFieldErr struct {
	Errors []string `json:"non_field_errors"`
}

func (err nonFieldErr) hasErr() bool { return len(err.Errors) > 0 }
func (err nonFieldErr) all() string  { return strings.Join(err.Errors, ", ") }

// AddContact creates a new Contact in Mojo
//
// Contact ID and GroupID must be provided. At least one contact field should be provided
//
// Return ErrDuplicate if a contact with same ID already exists. Return other errors
// if can't make the request of if Mojo returns an error
func (mj *Mojo) AddContact(contacts ...Contact) error {
	reqbody, err := json.Marshal(contacts)
	if err != nil {
		return &ErrInvalid{Msg: err.Error()}
	}
	url := prefixHTTP(mj.URL) + "/api/contacts/bulk_create/"
	resbody, err := mj.post(url, reqbody)
	if err != nil {
		return err
	}
	var data mojoResponse
	if err := json.Unmarshal(resbody, &data); err != nil {
		return fmt.Errorf("mojo: POST %s %s decoding %s (%v)", url, string(reqbody), string(resbody), err)
	}
	if data.isLockedError() {
		return fmt.Errorf("mojo: %s", data.errorMsg())
	}
	if data.isDuplicate() {
		return &ErrDuplicate{IDs: data.duplicatedIDs()}
	}
	if data.isError() {
		return &ErrInvalid{Msg: data.errorMsg()}
	}
	return nil
}

func (mj *Mojo) post(url string, reqbody []byte) (resbody []byte, err error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqbody))
	if err != nil {
		return []byte{}, fmt.Errorf("mojo: POST %s %s fail to build request (%v)", url, string(reqbody), err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mj.Token)
	if mj.HTTP == nil {
		mj.HTTP = &http.Client{Timeout: 3 * time.Second}
	}
	res, err := mj.HTTP.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("mojo: POST %s %s fail (%v)", url, string(reqbody), err)
	}
	defer res.Body.Close()
	resbody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("mojo: POST %s %s fail to read response (%v)", url, string(reqbody), err)
	}
	if res.StatusCode == 403 {
		return []byte{}, newForbidden(resbody)
	}
	if res.StatusCode == 400 {
		return []byte{}, &ErrInvalid{Msg: fmt.Sprintf("POST %s %s %d validation error %s", url, string(reqbody), res.StatusCode, string(resbody))}
	}
	if res.StatusCode != 200 {
		return []byte{}, fmt.Errorf("mojo: POST %s %s status %d with body %v", url, string(reqbody), res.StatusCode, string(resbody))
	}
	return resbody, nil
}

func newForbidden(body []byte) error {
	var data struct {
		Detail string `json:"detail"`
	}
	var msg string
	if err := json.Unmarshal(body, &data); err != nil || data.Detail == "" {
		msg = string(body)
	} else {
		msg = data.Detail
	}
	return &ErrForbidden{Msg: msg}
}

type mojoResponse struct {
	Errors                 []string `json:"errors"`
	DuplicatedAPIContactID []string `json:"duplicated_api_contact_id"`
}

func (resp mojoResponse) isError() bool {
	return len(resp.Errors) >= 1
}

func (resp mojoResponse) isLockedError() bool {
	return len(resp.Errors) == 1 && resp.Errors[0] == "Previous request was not finished or was interrupted."
}

func (resp mojoResponse) isDuplicate() bool {
	return resp.isError() && len(resp.DuplicatedAPIContactID) > 0
}

func (resp mojoResponse) errorMsg() string {
	return strings.Join(resp.Errors, " ")
}

func (resp mojoResponse) duplicatedIDs() []string {
	if !resp.isDuplicate() {
		return []string{}
	}
	return resp.DuplicatedAPIContactID
}

// Contact to be created in Mojo
//
// Either Name must provided OR at least one of Email, MobilePhone, WorkPhone, HomePhone
type Contact struct {
	ID                                string // required
	GroupID                           int    // required
	Name                              string
	Address, City, State, Zip         string
	Email                             string
	MobilePhone, WorkPhone, HomePhone string
	Notes                             []string
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
		City:    c.City,
		State:   c.State,
		Zip:     c.Zip,
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
	for _, nt := range c.Notes {
		cc.Notes = append(cc.Notes, note{1, nt})
	}
	data, err := json.Marshal(cc)
	return data, err
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
	City    string           `json:"city,omitempty"`
	State   string           `json:"state,omitempty"`
	Zip     string           `json:"zip_code,omitempty"`
	Notes   []note           `json:"contactnote_set,omitempty"`
	// list of phones and emails
	// 1-work, 2-mobile, 3-home, 4-email, 5-other
	Media []media `json:"mediainfo_set,omitempty"`
}

type media struct {
	Type  int    `json:"type"`
	Value string `json:"value"`
}

type note struct {
	Type     int    `json:"type"`
	Contents string `json:"contents"`
}

func prefixHTTP(domain string) string {
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return domain
	}
	return "https://" + domain
}
