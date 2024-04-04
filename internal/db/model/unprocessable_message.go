package model

const UnprocessableMsgCollection = "unprocessable_messages"

type UnprocessableMessageDocument struct {
	MessageBody string `bson:"message_body"`
	Recepit     string `bson:"recepit"`
}

func NewUnprocessableMessageDocument(messageBody, receipt string) *UnprocessableMessageDocument {
	return &UnprocessableMessageDocument{
		MessageBody: messageBody,
		Recepit:     receipt,
	}
}
