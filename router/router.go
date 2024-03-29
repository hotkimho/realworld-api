package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/hotkimho/realworld-api/controller/comment"

	"github.com/hotkimho/realworld-api/controller/article"
	"github.com/hotkimho/realworld-api/controller/auth"
	"github.com/hotkimho/realworld-api/controller/user"
	"github.com/hotkimho/realworld-api/env"
	"github.com/hotkimho/realworld-api/responder"
	"github.com/hotkimho/realworld-api/types"
)

type Router struct {
	Server     *mux.Router
	CorsHandle *cors.Cors
}

func (m *Router) Init() {
	m.Server = mux.NewRouter()

	m.InitSwagger()
	m.InitCORS()
	m.AddRoute(ArticleRouter)
	m.AddRoute(CommentRouter)
	m.AddRoute([][]*Route{
		{
			{
				Method:      "POST",
				Path:        "/user/signup",
				HandlerFunc: auth.SignUp,
			},
			{
				Method:      "POST",
				Path:        "/user/signin",
				HandlerFunc: auth.SignIn,
			},
			{
				Method:      "GET",
				Path:        "/heartbeat",
				HandlerFunc: auth.Heartbeat,
			},
			{
				Method:      "GET",
				Path:        "/user/{user_id}/profile",
				HandlerFunc: user.ReadUserProfile,
				Middleware:  []Middleware{UserAuthMiddleware},
			},
			{
				Method:      "PUT",
				Path:        "/user/{user_id}/profile",
				HandlerFunc: user.UpdateUserProfile,
				Middleware:  []Middleware{UserAuthMiddleware},
			},
			{
				Method:      "GET",
				Path:        "/user/{user_id}/token-refresh",
				HandlerFunc: auth.RefreshToken,
			},
			{
				Method:      "POST",
				Path:        "/user/verify-email",
				HandlerFunc: auth.VerifyEmail,
			},
			{
				Method:      "POST",
				Path:        "/user/verify-username",
				HandlerFunc: auth.VerifyUsername,
			},
		},
	})
}

var ArticleRouter = [][]*Route{
	{
		{
			Method:      "POST",
			Path:        "/user/{user_id}/article",
			HandlerFunc: article.CreateArticle,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
		{
			Method:      "GET",
			Path:        "/user/{user_id}/article/{article_id}",
			HandlerFunc: article.ReadArticleByID,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
		{
			Method:      "PUT",
			Path:        "/user/{user_id}/article/{article_id}",
			HandlerFunc: article.UpdateArticle,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
		{
			Method:      "DELETE",
			Path:        "/user/{user_id}/article/{article_id}",
			HandlerFunc: article.DeleteArticle,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
		{
			Method:      "GET",
			Path:        "/articles",
			HandlerFunc: article.ReadArticleByOffset,
		},
	},
}

var CommentRouter = [][]*Route{
	{
		{
			Method:      "POST",
			Path:        "/user/{user_id}/article/{article_id}/comment",
			HandlerFunc: comment.CreateComment,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
		{
			Method:      "GET",
			Path:        "/user/{user_id}/article/{article_id}/comments",
			HandlerFunc: comment.ReadComments,
		},
		{
			Method:      "PUT",
			Path:        "/user/{user_id}/article/{article_id}/comment/{comment_id}",
			HandlerFunc: comment.UpdateComment,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
		{
			Method:      "DELETE",
			Path:        "/user/{user_id}/article/{article_id}/comment/{comment_id}",
			HandlerFunc: comment.DeleteComment,
			Middleware:  []Middleware{UserAuthMiddleware},
		},
	},
}

func (m *Router) AddRoute(routeMaps [][]*Route) {
	for _, routeMap := range routeMaps {
		for _, route := range routeMap {

			handler := route.HandlerFunc

			// 미드웨어가 있는 경우
			for i := len(route.Middleware) - 1; i >= 0; i-- {
				handler = route.Middleware[i](handler)
			}

			m.Server.Handle(route.Path, handler).Methods(route.Method)
		}
	}
}

func (m *Router) InitSwagger() {
	fmt.Println(env.Config.Swagger.Host)
	m.Server.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL(env.Config.Swagger.Host), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)
}

// 5173
func (m *Router) InitCORS() {
	m.CorsHandle = cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"https://localhost:3000",
			"http://localhost:5173",
			"https://localhost:5173",
			"http://kp-realworld.com",
			"https://kp-realworld.com",
			"http://*.kp-realworld.com",
			"https://*.kp-realworld.com",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"Authorization",
			"X-CSRF-Token",
		},
		AllowCredentials: true,
	})

}

type Middleware func(handlerFunc http.HandlerFunc) http.HandlerFunc
type Route struct {
	Method      string
	Path        string
	HandlerFunc http.HandlerFunc
	Middleware  []Middleware
}

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// todo : logging

		next(w, r)
	}
}

func UserAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("Authorization")
		if token == "" {
			responder.ErrorResponse(w, http.StatusUnauthorized, "token is empty")
			return
		}

		parsedToken, err := jwt.ParseWithClaims(token, &types.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(env.Config.Auth.Secret), nil
		})
		if err != nil {
			// 토큰 만료
			if strings.Compare(err.Error(), types.ERR_EXPIRED_TOKEN) == 0 {
				responder.ErrorResponse(w, http.StatusUnauthorized, "Token expired")
				return
			}
			responder.ErrorResponse(w, http.StatusUnauthorized, "token is invalid")
			return
		}

		var userID int64
		if claim, ok := parsedToken.Claims.(*types.JWTClaims); ok && parsedToken.Valid {
			// 토큰의 ID 와 요청의 ID 가 일치하는지 확인
			if claim.UserID <= 0 {
				responder.ErrorResponse(w, http.StatusBadRequest, "token is invalid")
				return
			}
			userID = claim.UserID
			// 필요 시, api path의 user_id 와 토큰의 user_id 를 비교
			//vars := mux.Vars(r)
			//val, ok := vars["user_id"]
			//if !ok {
			//	responder.ErrorResponse(w, http.StatusBadRequest, "user_id is empty")
			//	return
			//}
			//
			//currentUserID, err := strconv.ParseInt(val, 10, 64)
			//if err != nil {
			//	responder.ErrorResponse(w, http.StatusBadRequest, "user_id is invalid")
			//	return
			//}
		} else {
			fmt.Println("err : ", err.Error())
			responder.ErrorResponse(w, http.StatusUnauthorized, "token is invalid")
			w.WriteHeader(401)
			return
		}

		ctx := context.WithValue(r.Context(), "ctx_user_id", userID)
		next(w, r.WithContext(ctx))
	}
}

func testMiddlewareA(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("a")

		next(w, r)
	}
}

func testMiddlewareB(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("b")
		next(w, r)
	}

}

func testMiddlewareC(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("c")
		next(w, r)
	}
}
