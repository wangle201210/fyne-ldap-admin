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
	"time"
)

type LdapAdmin struct {
	fyne.App
	sync.Mutex
	windows       fyne.Window
	resultWindow  fyne.Window
	resultContent fyne.CanvasObject
	statusLabel   *widget.Label
	currentList   *widget.List
	labels        [][]*canvas.Text
	wg            *sync.WaitGroup
	ldapPool      *dao.LDAPPool
	detailContent *widget.TextGrid
	search
}

type search struct {
	configAccordionItem *widget.AccordionItem
	ldapConn            *config.LdapConf
	result              *fyne.Container
	searchReq           *dao.SearchReq
	pageControl         *ldap.ControlPaging
	conn                *ldap.Conn
	data                []*ldap.Entry // 搜索到的结果
	selectData          *ldap.Entry   // 需要被显示的data
}

func NewApp(app fyne.App) *LdapAdmin {
	res := &LdapAdmin{
		App:     app,
		windows: app.NewWindow(config.AppID),
	}
	res.App.Settings().BuildType()
	res.initConfig()

	// Initialize LDAP connection pool
	server, _ := res.ldapConn.Addr.Get()
	port, _ := res.ldapConn.Port.Get()
	username, _ := res.ldapConn.Username.Get()
	password, _ := res.ldapConn.Password.Get()
	
	pool, err := dao.NewLDAPPool(&dao.LDAPConfig{
		Server:   server,
		Port:     port,
		Username: username,
		Password: password,
		PoolSize: 5,
		Timeout:  30 * time.Second,
	})
	
	if err != nil {
		log.Errorf("Failed to create LDAP pool: %v", err)
	} else {
		res.ldapPool = pool
	}

	app.Lifecycle().SetOnStopped(func() {
		res.ldapConn.Save() // Save data on exit
		if res.ldapPool != nil {
			// Close all connections in pool
			for i := 0; i < 5; i++ {
				if conn, err := res.ldapPool.GetConnection(); err == nil {
					conn.Close()
				}
			}
		}
	})

	res.searchReq = dao.NewSearchReq()
	res.result = container.NewVBox()
	
	// Layout
	content := container.NewVBox(
		res.MainPanel(),
	)

	res.windows.SetContent(content)
	return res
}

func (x *LdapAdmin) initConfig() {
	x.ldapConn = config.InitLdapCon()
	x.initConfigPanel()
}

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
		// x.result,
	)
	return content
}

func (x *LdapAdmin) GetConn() (conn *ldap.Conn) {
	if x.ldapPool != nil {
		conn, err := x.ldapPool.GetConnection()
		if err == nil {
			return conn
		}
		log.Errorf("Failed to get connection from pool: %v", err)
	}
	
	// Fallback to direct connection
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
	if ldapConn == nil {
		x.result.Add(widget.NewLabel("Failed to establish LDAP connection"))
		return
	}
	
	defer func() {
		if x.ldapPool != nil {
			x.ldapPool.ReleaseConnection(ldapConn)
		}
	}()

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
		x.result.Add(widget.NewLabel("No more pages available"))
		return
	}

	// Define attributes to retrieve
	attributes := []string{
		"cn", "sn", "givenName",
		"mail", "telephoneNumber",
		"uid", "uidNumber", "gidNumber",
		"o", "ou", "title",
		"objectClass", "createTimestamp", "modifyTimestamp",
	}

	entries, err := dao.Search(ldapConn, baseDN, filter, x.search.pageControl, attributes)
	if err != nil {
		log.Errorf("Search failed: %v", err)
		x.result.Add(widget.NewLabel(fmt.Sprintf("Search failed: %v", err)))
		return
	}

	x.data = entries
	x.ResultShow()
}

func (x *LdapAdmin) Run() {
	x.windows.Resize(fyne.NewSize(800, 800))
	x.windows.ShowAndRun()
}
