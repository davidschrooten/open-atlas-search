package cluster

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

// ClusterServiceServer implements the gRPC cluster service
type ClusterServiceServer struct {
	manager *Manager
}

// NewClusterServiceServer creates a new gRPC service server
func NewClusterServiceServer(manager *Manager) *ClusterServiceServer {
	return &ClusterServiceServer{
		manager: manager,
	}
}

// JoinCluster handles requests from nodes wanting to join the cluster
func (s *ClusterServiceServer) JoinCluster(ctx context.Context, req *JoinRequest) (*JoinResponse, error) {
	log.Printf("Node %s requesting to join cluster from %s", req.NodeId, req.Address)
	
	// Add the node to the cluster (simplified implementation)
	s.manager.AddNode(req.NodeId, req.Address)
	
	return &JoinResponse{
		Message: fmt.Sprintf("Node %s successfully joined the cluster", req.NodeId),
	}, nil
}

// GetClusterState returns the current cluster state
func (s *ClusterServiceServer) GetClusterState(ctx context.Context, req *StateRequest) (*StateResponse, error) {
	nodeIds := s.manager.GetNodeIDs()
	
	return &StateResponse{
		NodeIds: nodeIds,
	}, nil
}

// StartGRPCServer starts the gRPC server for cluster communication
func (m *Manager) StartGRPCServer(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	clusterService := NewClusterServiceServer(m)
	
	// Register the service (we'll need to implement the registration manually)
	// RegisterClusterServiceServer(grpcServer, clusterService)
	
	log.Printf("Starting gRPC server on port %d", port)
	
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server failed: %v", err)
		}
	}()
	
	m.grpcServer = grpcServer
	return nil
}

// StopGRPCServer stops the gRPC server
func (m *Manager) StopGRPCServer() {
	if m.grpcServer != nil {
		m.grpcServer.GracefulStop()
	}
}

// JoinRequest represents a request to join the cluster
type JoinRequest struct {
	NodeId  string `json:"node_id"`
	Address string `json:"address"`
}

// JoinResponse represents a response to a join request
type JoinResponse struct {
	Message string `json:"message"`
}

// StateRequest represents a request for cluster state
type StateRequest struct{}

// StateResponse represents cluster state information
type StateResponse struct {
	NodeIds []string `json:"node_ids"`
}
