package app

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/martian/log"
	"github.com/wangle201210/fyne-ldap-admin/app/dao"
	"github.com/wangle201210/fyne-ldap-admin/config"
	"sync"
)

type LdapAdmin struct {
	app fyne.App
	sync.Mutex
	windows  fyne.Window
	payTable fyne.Window
	history  fyne.Window
	labels   [][]*canvas.Text
	wg       *sync.WaitGroup
	search
}

type search struct {
	configAccordionItem *widget.AccordionItem
	ldapConn            *config.LdapConf
	result              *fyne.Container
	searchReq           *dao.SearchReq
	pageControl         *ldap.ControlPaging
	conn                *ldap.Conn
}

func NewApp(app fyne.App) *LdapAdmin {
	res := &LdapAdmin{
		app:     app,
		windows: app.NewWindow(config.AppID),
	}

	res.initConfig()
	app.Lifecycle().SetOnStopped(func() {
		res.ldapConn.Save() // 退出时保存数据
	})
	res.searchReq = dao.NewSearchReq()
	res.result = container.NewVBox()
	// 布局
	content := container.NewVBox(
		res.MainPanel(),
	)

	res.windows.SetContent(content)
	return res
}

// initConfigPanel 配置面板
func (x *LdapAdmin) initConfigPanel() {
	serverEntry := widget.NewEntryWithData(x.ldapConn.Addr)
	serverEntry.SetPlaceHolder("LDAP Server Address")
	portEntry := widget.NewEntryWithData(x.ldapConn.Port)
	portEntry.SetPlaceHolder("LDAP Server Port")

	usernameEntry := widget.NewEntryWithData(x.ldapConn.Username)
	usernameEntry.SetPlaceHolder("Username")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.Bind(x.ldapConn.Password)
	passwordEntry.SetPlaceHolder("Password")

	limitEntry := widget.NewEntryWithData(binding.IntToString(x.ldapConn.Limit))
	limitEntry.SetPlaceHolder("限制返回条数")

	configAccordionItem := &widget.AccordionItem{
		Title: "配置",
		Detail: container.NewVBox(
			serverEntry,
			portEntry,
			usernameEntry,
			passwordEntry,
			limitEntry,
		),
		Open: true,
	}
	x.configAccordionItem = configAccordionItem
}

// MainPanel 主面板
func (x *LdapAdmin) MainPanel() *fyne.Container {

	baseDNEntry := widget.NewEntryWithData(x.searchReq.BaseDN)
	baseDNEntry.SetPlaceHolder("Base DN")

	filterEntry := widget.NewEntryWithData(x.searchReq.Filter)
	filterEntry.SetPlaceHolder("Search Filter (e.g., (objectClass=*))")

	searchButton := widget.NewButton("Search", x.Search)
	accordion := widget.NewAccordion(
		x.configAccordionItem,
	)
	content := container.NewVBox(
		accordion,
		baseDNEntry,
		filterEntry,
		searchButton,
		x.result,
	)
	return content
}

func (x *LdapAdmin) initConfig() {
	x.ldapConn = config.InitLdapCon()
	x.initConfigPanel()
}

func (x *LdapAdmin) Run() {
	x.windows.Resize(fyne.NewSize(400, 400))
	x.windows.ShowAndRun()
}

func (x *LdapAdmin) GetConn() (conn *ldap.Conn) {
	if x.conn != nil {
		return x.conn
	}
	x.conn, _ = x.GetLdapConn()
	return x.conn
}

func (x *LdapAdmin) GetLdapConn() (conn *ldap.Conn, err error) {
	// saveConf
	server, _ := x.ldapConn.Addr.Get()
	port, _ := x.ldapConn.Port.Get()
	username, _ := x.ldapConn.Username.Get()
	password, _ := x.ldapConn.Password.Get()

	conn, err = dao.GetLdap(server, port, username, password)
	if err != nil {
		x.result.Add(widget.NewLabel(fmt.Sprintf("Connection failed: %v", err)))
		return
	}
	return
}

func (x *LdapAdmin) Search() {
	x.doSearch(true)
}

func (x *LdapAdmin) SearchNextPage() {
	x.doSearch(false)
}

func (x *LdapAdmin) doSearch(isFirst bool) {
	x.configAccordionItem.Open = false
	x.windows.Content().Refresh()
	x.result.RemoveAll()
	ldapConn := x.GetConn()
	// defer ldapConn.Close()
	baseDN, _ := x.searchReq.BaseDN.Get()
	filter, _ := x.searchReq.Filter.Get()
	limit, _ := x.ldapConn.Limit.Get()
	if limit == 0 || limit > 100 {
		limit = 1000
	}
	if isFirst {
		x.search.pageControl = ldap.NewControlPaging(uint32(limit))
	}
	if x.search.pageControl == nil {
		x.result.Add(widget.NewLabel("没有下一页了"))
		return
	}
	entries, err := dao.Search(ldapConn, baseDN, filter, x.search.pageControl)
	if err != nil {
		log.Errorf("Search failed: %v", err)
		x.result.Add(widget.NewLabel(fmt.Sprintf("Search failed: %v", err)))
		return
	}
	// 显示结果
	result := fmt.Sprintf("Total: %d\n\n", len(entries))
	for _, entry := range entries {
		result += fmt.Sprintf("dn: %s\n", entry.DN)
		for _, e := range entry.Attributes {
			result += fmt.Sprintf("%s: %s\n", e.Name, e.Values)
		}
		result += "\n"
	}
	fmt.Printf("%s\n", result)
	size := fyne.Size{Width: 400, Height: 400}
	resultLabel := container.NewVBox(widget.NewLabel(result))
	// 搜索按钮
	nextButton := widget.NewButton("下一页", x.SearchNextPage)
	scroll := container.NewScroll(resultLabel)
	scroll.SetMinSize(size)
	x.result.Add(nextButton)
	x.result.Add(scroll)
	x.result.Refresh()
}
