package builtin

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatedier/frp/pkg/plugin/server"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type DynamicConfigDistributionManager struct {
	db *gorm.DB
}

func NewDynamicConfigDistributionManager() *DynamicConfigDistributionManager {
	client, err := gorm.Open(sqlite.Open(getDBPath()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = client.AutoMigrate(&User{}); err != nil {
		fmt.Println("failed migrate user")
		os.Exit(1)
	}
	return &DynamicConfigDistributionManager{
		db: client,
	}
}

func (p *DynamicConfigDistributionManager) Name() string {
	return "dynamic_config_distribution_manager"
}

func (p *DynamicConfigDistributionManager) IsSupport(op string) bool {
	return op == "Login"
}

func (p *DynamicConfigDistributionManager) Handle(ctx context.Context, op string, content any) (res *server.Response, retContent any, err error) {
	fmt.Printf("%s %+v\n", op, content)
	switch op {
	case "Login":
		if loginReq, ok := content.(server.LoginContent); ok {
			fmt.Println(loginReq.User)
			if _, err := p.getUserByName(loginReq.User); err != nil {
				return &server.Response{
					Reject:       true,
					RejectReason: fmt.Sprintf("client not found"),
					Unchange:     true,
					Content:      content,
				}, content, nil
			}
			return &server.Response{
				Reject:       false,
				RejectReason: "",
				Unchange:     false,
				Content:      &loginReq,
			}, &loginReq, nil
		}
	}
	return &server.Response{
		Reject:       true,
		RejectReason: fmt.Sprintf("not support option %s", op),
		Unchange:     true,
		Content:      content,
	}, content, nil
}

func (p *DynamicConfigDistributionManager) HandleGetConfig(writer http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)
	token := params["token"]
	u, err := p.getUserByName(token)
	if err != nil {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(u.Config))
}

func (p *DynamicConfigDistributionManager) HandleSetConfig(writer http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)
	token := params["token"]

	body, err := io.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	if err = p.UpdateConfigByUser(token, string(body)); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	writer.WriteHeader(http.StatusOK)
	return
}

func (p *DynamicConfigDistributionManager) UpdateConfigByUser(user string, config string) error {
	p.db.Delete(&User{Token: user})

	return p.db.Create(&User{
		Token:  user,
		Config: config,
	}).Error
}

func (p *DynamicConfigDistributionManager) getUserByName(user string) (*User, error) {
	var u User
	if err := p.db.Order("created_at DESC").First(&u, "token = ?", user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client not found")
		}
		return nil, fmt.Errorf("failed to find client")
	}
	return &u, nil
}

type User struct {
	Token  string `gorm:"index"`
	Config string

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

type FrpLogin struct {
	User          string            `json:"user"`
	PrivilegeKey  string            `json:"privilege_key"`
	RunId         string            `json:"run_id"`
	Metas         map[string]string `json:"metas"`
	ClientAddress string            `json:"client_address"`
}

type FrpLoginRequest struct {
	Content FrpLogin `json:"content"`
}

func getDBPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "frps.db")
}
