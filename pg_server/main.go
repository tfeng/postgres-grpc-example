package main

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-pg/pg"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/tfeng/postgres-grpc-example/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"time"
)

func auth(ctx context.Context) (context.Context, error) {
	if tokenString, err := grpc_auth.AuthFromMD(ctx, "bearer"); err != nil {
		return ctx, nil
	} else {
		if token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey.Public(), nil
		}); err != nil {
			logger.Error("Unable to parse token", zap.Error(err))
			return ctx, err
		} else {
			if err := token.Claims.Valid(); err != nil {
				logger.Error("Invalid token", zap.Error(err))
				return ctx, err
			} else {
				return context.WithValue(ctx, "claims", token.Claims), nil
			}
		}
	}
}

func connect() *pg.DB {
	return pg.Connect(&pg.Options{
		Addr:     "localhost:5432",
		User:     "postgres",
		Password: "password",
	})
}

func createTable() error {
	return db.CreateTable(&models.User{}, nil)
}

func dropTable() error {
	return db.DropTable(&models.User{}, nil)
}

func generateKey() *rsa.PrivateKey {
	if key, err := rsa.GenerateKey(rand.Reader, 1024); err != nil {
		logger.Fatal("Unable to generate RSA key", zap.Error(err))
		return nil
	} else {
		return key
	}
}

func generateToken(user models.User) (string, error) {
	now := jwt.TimeFunc()
	claims := jwt.MapClaims{
		"id":   user.Id,
		"name": user.Name,
		"role": user.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func initialize() {
	if err := dropTable(); err != nil {
		logger.Info("Unable to drop table", zap.Error(err))
	}

	if err := createTable(); err != nil {
		logger.Fatal("Unable to create table. ", zap.Error(err))
		return
	}
}

func startService() error {
	listener, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		logger.Fatal("Unable to start service", zap.Error(err))
		return err
	}

	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_auth.StreamServerInterceptor(auth),
			grpc_validator.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(logger))),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_auth.UnaryServerInterceptor(auth),
			grpc_validator.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(logger))))
	models.RegisterUserServiceServer(s, &UserService{})

	reflection.Register(s)
	return s.Serve(listener)
}

type UserService struct{}

func (userService *UserService) Create(ctx context.Context, request *models.CreateRequest) (*models.CreateResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Unable to generate hashed password", zap.Error(err))
		return nil, err
	}

	user := models.User{Name: request.Name, HashedPassword: string(hashedPassword), Role: request.Role}
	if err := db.Insert(&user); err != nil {
		logger.Error("Unable to insert user", zap.Error(err))
		return nil, err
	} else {
		return &models.CreateResponse{user.Id}, nil
	}
}

func (userService *UserService) Get(ctx context.Context, request *models.GetRequest) (*models.User, error) {
	if claims, ok := ctx.Value("claims").(jwt.Claims); !ok {
		logger.Error("User is not authorized")
		return nil, errors.New("not authorized")
	} else {
		mapClaims := claims.(jwt.MapClaims)
		user := models.User{Id: int64(mapClaims["id"].(float64))}
		if err := db.Select(&user); err != nil {
			logger.Error("Unable to find user", zap.Error(err))
			return nil, err
		} else {
			user.HashedPassword = ""
			return &user, nil
		}
	}
}

func (userService *UserService) Login(ctx context.Context, request *models.LoginRequest) (*models.LoginResponse, error) {
	user := models.User{Id: request.Id}
	if err := db.Select(&user); err != nil {
		logger.Error("Unable to find user", zap.Error(err))
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(request.Password)); err != nil {
		logger.Error("Wrong password", zap.Error(err))
		return nil, err
	}

	if tokenString, err := generateToken(user); err != nil {
		logger.Error("Unable to generate token", zap.Error(err))
		return nil, err
	} else {
		return &models.LoginResponse{Token: tokenString}, nil
	}
}

var (
	db        = connect()
	logger, _ = zap.NewDevelopment()
	secretKey = generateKey()
)

func main() {
	initialize()
	startService()
}
