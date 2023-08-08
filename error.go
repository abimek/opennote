package main

type WebsiteRequestError int

const (
	NonExistentUser WebsiteRequestError = iota
	InvalidRequestContent
	FirestoreError
	ServerError
	PineconeError
	InvalidCredsError
	UserExistsError
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
	case InvalidCredsError:
		return "InvalidCredsError"
	case UserExistsError:
		return "UserExistsError"
	}
	return ""
}
