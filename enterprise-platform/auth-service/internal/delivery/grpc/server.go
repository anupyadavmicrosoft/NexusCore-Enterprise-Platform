package grpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/nexuscore/auth-service/internal/domain"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GrpcIdentityServer implements gRPC internal identity contract
type GrpcIdentityServer struct {
	usecase domain.IdentityUsecase
}

func NewGrpcIdentityServer(uc domain.IdentityUsecase) *GrpcIdentityServer {
	return &GrpcIdentityServer{usecase: uc}
}

// StartGrpcServer runs the TCP gRPC listener
func (s *GrpcIdentityServer) StartGrpcServer(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("Failed to bind TCP listener for gRPC Server", "addr", addr, "error", err)
		return err
	}

	grpcServer := grpc.NewServer()
	
	// Normally we would register the protobuf compilation:
	// pb.RegisterIdentityServiceServer(grpcServer, s)
	
	slog.Info("gRPC Microservice Core Server successfully listening", "addr", addr)
	
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server listener crashed", "error", err)
		}
	}()

	return nil
}

// -----------------------------------------------------------------
// DIRECT RPC METHOD ENDPOINTS (SIMULATED gRPC HANDLERS)
// -----------------------------------------------------------------

func (s *GrpcIdentityServer) VerifyToken(ctx context.Context, token string) (*domain.TokenClaims, error) {
	tr := otel.Tracer("grpc-delivery")
	ctx, span := tr.Start(ctx, "gRPC.VerifyToken")
	defer span.End()

	slog.Info("gRPC RPC call received: VerifyToken")
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "authorization bearer token string cannot be empty")
	}

	claims, err := s.usecase.VerifyToken(ctx, token)
	if err != nil {
		slog.Warn("gRPC VerifyToken unauthorized error context", "error", err)
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return claims, nil
}

func (s *GrpcIdentityServer) CheckAuthorization(ctx context.Context, req *domain.AuthzCheckRequest) (*domain.AuthzCheckResponse, error) {
	tr := otel.Tracer("grpc-delivery")
	ctx, span := tr.Start(ctx, "gRPC.CheckAuthorization")
	defer span.End()

	slog.Info("gRPC RPC call received: CheckAuthorization", "userID", req.UserID, "resource", req.Resource)
	
	if req.UserID == "" || req.Resource == "" || req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "userID, resource, and action are mandatory query scopes")
	}

	res, err := s.usecase.EvaluateAuthorization(ctx, *req)
	if err != nil {
		slog.Error("gRPC dynamic evaluation processing failure", "error", err)
		return nil, status.Error(codes.Internal, "dynamic policy processing crashed")
	}

	return res, nil
}
