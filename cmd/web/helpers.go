package main

//send email dengan param Message
//helper biar ga kelupaan buat increment waitgroup nya
//channel MailerChan bakal diterima di app.listenForEmail()
func (app *Config) sendMail(msg Message) {
	app.Wg.Add(1)

	app.Mailer.MailerChan <- msg
}