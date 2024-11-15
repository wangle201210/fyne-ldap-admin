package main

import (
	fapp "fyne.io/fyne/v2/app"
	"github.com/wangle201210/fyne-ldap-admin/app"
	"github.com/wangle201210/fyne-ldap-admin/config"
)

func main() {
	myApp := fapp.NewWithID(config.AppID)
	ldapAdmin := app.NewApp(myApp)
	ldapAdmin.Run()
}
