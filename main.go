package main

import (
	"crypto/tls"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-ldap/ldap/v3"
)

// LDAP连接函数
func connectToLDAP(server string, port int, username, password string) (*ldap.Conn, error) {
	l, err := ldap.DialURL(fmt.Sprintf("%s:%d", server, port), ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))

	if err != nil {
		return nil, err
	}

	_, err = l.SimpleBind(&ldap.SimpleBindRequest{
		Username: username, // cn=admin,dc=sicau,dc=edu,dc=cn
		Password: password, // admin
	})
	if err != nil {
		return nil, err
	}
	return l, nil
}

// 查询LDAP数据
func searchLDAP(l *ldap.Conn, baseDN, filter string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"dn", "cn", "mail"}, // 您可以根据需要调整属性
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return sr.Entries, nil
}

func main() {
	// 初始化Fyne应用
	myApp := app.New()
	myWindow := myApp.NewWindow("LDAP Management Tool")

	// 用户名、密码和服务器输入框
	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("LDAP Server Address")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("LDAP Server Port")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	baseDNEntry := widget.NewEntry()
	baseDNEntry.SetPlaceHolder("Base DN")

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Search Filter (e.g., (objectClass=*))")
	filterEntry.SetText("(objectClass=*)")

	resultLabel := widget.NewLabel("")

	// 搜索按钮
	searchButton := widget.NewButton("Search", func() {
		server := serverEntry.Text
		port := 389 // 默认端口
		fmt.Sscanf(portEntry.Text, "%d", &port)
		username := usernameEntry.Text
		password := passwordEntry.Text
		baseDN := baseDNEntry.Text
		filter := filterEntry.Text

		conn, err := connectToLDAP(server, port, username, password)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("Connection failed: %v", err))
			return
		}
		defer conn.Close()

		entries, err := searchLDAP(conn, baseDN, filter)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("Search failed: %v", err))
			return
		}

		// 显示结果
		result := "Results:\n"
		for _, entry := range entries {
			result += fmt.Sprintf("DN: %s\nCN: %s\nMail: %s\n\n", entry.DN, entry.GetAttributeValue("cn"), entry.GetAttributeValue("mail"))
		}
		resultLabel.SetText(result)
	})

	// 布局
	content := container.NewVBox(
		serverEntry,
		portEntry,
		usernameEntry,
		passwordEntry,
		baseDNEntry,
		filterEntry,
		searchButton,
		resultLabel,
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 400))
	myWindow.ShowAndRun()
}
