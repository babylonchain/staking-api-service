package model

const UnprocessableMsgCollection = "unprocessable_messages"

type UnprocessableMessageDocument struct {
	MessageBody string `bson:"message_body"`
	Receipt     string `bson:"receipt"`
}

func NewUnprocessableMessageDocument(messageBody, receipt string) *UnprocessableMessageDocument {
	return &UnprocessableMessageDocument{
		MessageBody: messageBody,
		Receipt:     receipt,
	}
}
