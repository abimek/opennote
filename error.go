package main

type WebsiteRequestError int

const (
	NonExistentUser WebsiteRequestError = iota
	InvalidRequestContent
	FirestoreError
	ServerError
	PineconeError
)

func (c WebsiteRequestError) String() string {
	switch c {
	case NonExistentUser:
		return "NonExistentUser"
	case InvalidRequestContent:
		return "InvalidRequestContent"
	case FirestoreError:
		return "FirestoreError"
	case ServerError:
		return "ServerError"
	case PineconeError:
		return "PineconeError"
	}
	return ""
}
