package handlers

import (
	"embed"
	"html/template"
	"net/http"
)

type Other struct {
	templates *embed.FS
}

func NewOther(
	templates *embed.FS,
) *Other {
	return &Other{
		templates,
	}
}

func (o *Other) WrongAccountPage(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("wrong-account-page").ParseFS(*o.templates, "templates/wrong_account.html"))
	err := t.ExecuteTemplate(w, "wrong_account.html", nil)
	if err != nil {
		http.Error(w, "An Unknown Error Has Occured, Please Try Again Later", http.StatusInternalServerError)
		return
	}
}
