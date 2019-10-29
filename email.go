package main

import (
	"errors"
	// "mime"
	"net/smtp"
)

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (proto string, toServer []byte, err error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) (toServer []byte, err error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unkown fromServer")
		}
	}
	return nil, nil
}

func sendEmail(to []string, subject string, msg string) error {
	auth := LoginAuth(emailAuthUser, emailAuthPass)

	// dec := new(mime.WordDecoder)
	// emailMsg := dec.Decode(hhjj
	emailMsg := []byte("To: " + to[0] + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		msg)

	err := smtp.SendMail(emailHostPort, auth, emailFrom, to, emailMsg)
	return err
}
