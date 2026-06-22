package service

import (
	"context"
	"net/http"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/imagestore"
	"momoko/pkg/response"
)

type ImageGenService struct {
	v1.UnimplementedSub2APIImageGenServer

	uc *biz.ImageGenUsecase
}

func NewImageGenService(uc *biz.ImageGenUsecase) *ImageGenService {
	return &ImageGenService{uc: uc}
}

// RegisterImageServer 注册公开图片服务（raw bytes，无 JSON 包裹）。
// 路径带 {id} 段以匹配 /images/<id>；gorilla/mux 支持 {name} 占位符。
func (s *ImageGenService) RegisterImageServer(srv *khttp.Server) {
	srv.HandleFunc(biz.PreImageGenImage+"/{id}", s.serveImage)
}

// authFromContext 从请求头取出 sub2api token 与 src_host。
func (s *ImageGenService) authFromContext(ctx context.Context) (token, srcHost string, err error) {
	r, ok := khttp.RequestFromServerContext(ctx)
	if !ok {
		return "", "", biz.ErrImageGenTokenInvalid
	}
	token = strings.TrimSpace(r.Header.Get("X-Sub2API-Token"))
	srcHost = strings.TrimSpace(r.Header.Get("X-Sub2API-Host"))
	if token == "" {
		return "", "", biz.ErrImageGenTokenInvalid
	}
	return token, srcHost, nil
}

func (s *ImageGenService) ListImageGenApiKeys(ctx context.Context, _ *v1.ListImageGenApiKeysRequest) (*v1.ListImageGenApiKeysResponse, error) {
	token, srcHost, err := s.authFromContext(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := s.uc.ListApiKeys(ctx, token, srcHost)
	if err != nil {
		return nil, err
	}
	return &v1.ListImageGenApiKeysResponse{ApiKeys: keys}, nil
}

func (s *ImageGenService) ListImageGenModels(ctx context.Context, req *v1.ListImageGenModelsRequest) (*v1.ListImageGenModelsResponse, error) {
	token, srcHost, err := s.authFromContext(ctx)
	if err != nil {
		return nil, err
	}
	models, err := s.uc.ListModels(ctx, token, srcHost, req.GetApiKeyId())
	if err != nil {
		return nil, err
	}
	return &v1.ListImageGenModelsResponse{Models: models}, nil
}

func (s *ImageGenService) ListImageGenGenerations(ctx context.Context, req *v1.ListImageGenGenerationsRequest) (*v1.ListImageGenGenerationsResponse, error) {
	token, srcHost, err := s.authFromContext(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.uc.ListGenerations(ctx, token, srcHost, int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	return &v1.ListImageGenGenerationsResponse{Generations: list}, nil
}

func (s *ImageGenService) CreateImageGenGeneration(ctx context.Context, req *v1.CreateImageGenGenerationRequest) (*v1.ImageGenGenerationResponse, error) {
	token, srcHost, err := s.authFromContext(ctx)
	if err != nil {
		return nil, err
	}
	gen, err := s.uc.CreateImageGenGeneration(ctx, token, srcHost, req)
	if err != nil {
		return nil, err
	}
	return &v1.ImageGenGenerationResponse{Generation: gen}, nil
}

func (s *ImageGenService) GetImageGenGeneration(ctx context.Context, req *v1.GetImageGenGenerationRequest) (*v1.ImageGenGenerationResponse, error) {
	token, srcHost, err := s.authFromContext(ctx)
	if err != nil {
		return nil, err
	}
	gen, err := s.uc.GetGeneration(ctx, token, srcHost, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.ImageGenGenerationResponse{Generation: gen}, nil
}

func (s *ImageGenService) DeleteImageGenGeneration(ctx context.Context, req *v1.DeleteImageGenGenerationRequest) (*v1.DeleteImageGenGenerationResponse, error) {
	token, srcHost, err := s.authFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DeleteGeneration(ctx, token, srcHost, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteImageGenGenerationResponse{}, nil
}

// serveImage 公开图片服务：/api/v1/public/sub2api/imagine/images/{id}
func (s *ImageGenService) serveImage(w khttp.ResponseWriter, r *khttp.Request) {
	id := strings.TrimPrefix(r.URL.Path, biz.PreImageGenImage)
	id = strings.TrimPrefix(id, "/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	img, err := s.uc.ServeImage(r.Context(), id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	fs, info, err := imagestore.Open(img.Path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer fs.Close()
	w.Header().Set("Content-Type", img.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, info.Name(), info.ModTime(), fs)
}
