package user

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jwjhuang/blog/service/app/logger"
	"github.com/jwjhuang/blog/service/model"
	"github.com/jwjhuang/blog/service/utils/auth"
	"github.com/jwjhuang/blog/service/utils/errs"
	"golang.org/x/crypto/bcrypt"
)

func newUser() IUserCenter {
	return &userUseCase{}
}

//IUserCenter define user's capabilities
type IUserCenter interface {
	Login(c *gin.Context, user *model.User) (*model.Token, error)
	Register(c *gin.Context, user *model.User) error
	Update(c *gin.Context, user *model.User) error
	GetUserByEmail(c *gin.Context, email string) (*model.UserResponse, error)
	GetProfile(c *gin.Context, account, email string) (*model.ProfileResponse, error)
	FollowUser(c *gin.Context, account, email string) (*model.ProfileResponse, error)
	UnFollowUser(c *gin.Context, account, email string) (*model.ProfileResponse, error)
}

type userUseCase struct {
}

func (uc *userUseCase) Login(c *gin.Context, user *model.User) (*model.Token, error) {

	dbUser, err := dao.User.GetUserByEmail(packet.DB, user.Email)
	if err != nil {
		logger.Log().Error(err)
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		logger.Log().Error(err)
		return nil, err
	}

	jwtToken, err := auth.GenerateJWTToken(dbUser, auth.SecretKey)
	if err != nil {
		logger.Log().Error(err)
		return nil, err
	}
	expiresIn := time.Hour * time.Duration(1)

	token := &model.Token{
		AccessToken: jwtToken,
		TokenType:   "bearer",
		ExpiresIN:   int(expiresIn.Seconds()),
	}

	return token, nil
}

func (uc *userUseCase) Register(c *gin.Context, user *model.User) error {
	if err := validateRegister(user); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hash)

	if err := dao.User.Insert(packet.DB, user); err != nil {
		return err
	}

	return nil
}

func (uc *userUseCase) Update(c *gin.Context, user *model.User) error {
	dbUser, err := dao.User.GetUserByEmail(packet.DB, user.Email)
	if err != nil {
		logger.Log().Error(err)
		return err
	}

	updateData, err := prepareUpdate(dbUser, user)
	if err != nil {
		logger.Log().Error(err)
		return err
	}

	if err := dao.User.Update(packet.DB, updateData); err != nil {
		logger.Log().Error(err)
		return err
	}

	return nil
}

func (uc *userUseCase) GetUserByEmail(c *gin.Context, email string) (*model.UserResponse, error) {
	resp, err := dao.User.GetUserByEmail(packet.DB, email)
	if err != nil {
		return nil, err
	}

	return resp.GeUserResponse(), nil
}

func (uc *userUseCase) GetProfile(c *gin.Context, name, email string) (*model.ProfileResponse, error) {
	me, err := dao.User.GetUserByEmail(packet.DB, email)
	if err != nil {
		return nil, err
	}

	view, err := dao.User.GetUserByCondition(packet.DB, &model.User{Username: name})
	if err != nil {
		return nil, err
	}

	isFollowing := dao.Profile.IsFollowing(packet.DB, view.ID, me.ID)

	return view.GeProfileResponse(isFollowing), nil
}

func (uc *userUseCase) FollowUser(c *gin.Context, name, email string) (*model.ProfileResponse, error) {
	me, err := dao.User.GetUserByEmail(packet.DB, email)
	if err != nil {
		return nil, err
	}

	view, err := dao.User.GetUserByCondition(packet.DB, &model.User{Username: name})
	if err != nil {
		return nil, err
	}

	if err := dao.Profile.FollowUser(packet.DB, view.ID, me.ID); err != nil {
		return nil, err
	}
	logger.Log().Infof("name=%v, email=%v", name, email)

	isFollowing := dao.Profile.IsFollowing(packet.DB, view.ID, me.ID)
	if isFollowing != true {
		return nil, errors.New("follow user failed")
	}

	return view.GeProfileResponse(isFollowing), nil
}

func (uc *userUseCase) UnFollowUser(c *gin.Context, name, email string) (*model.ProfileResponse, error) {
	me, err := dao.User.GetUserByEmail(packet.DB, email)
	if err != nil {
		return nil, err
	}

	view, err := dao.User.GetUserByCondition(packet.DB, &model.User{Username: name})
	if err != nil {
		return nil, err
	}

	if err := dao.Profile.UnFollowUser(packet.DB, view.ID, me.ID); err != nil {
		return nil, err
	}
	logger.Log().Infof("name=%v, email=%v", name, email)

	isFollowing := dao.Profile.IsFollowing(packet.DB, view.ID, me.ID)
	if isFollowing != false {
		return nil, errors.New("unfollow user failed")
	}

	return view.GeProfileResponse(isFollowing), nil
}

func validateRegister(user *model.User) error {
	if user.Email == "" {
		return errs.ErrInvalidRequest
	}

	if user.Username == "" {
		return errs.ErrInvalidRequest
	}

	if user.Password == "" {
		return errs.ErrInvalidRequest
	}

	return nil
}

func prepareUpdate(old, new *model.User) (*model.User, error) {
	if new.Username != "" {
		old.Username = new.Username
	}

	if new.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(new.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		old.Password = string(hash)
	}

	if new.Bio != "" {
		old.Bio = new.Bio
	}

	if new.Image != nil {
		old.Image = new.Image
	}

	return old, nil
}
