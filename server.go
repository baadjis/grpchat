package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/baadjis/grpchat/chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// The port the server is listening on.
const (
	port        = ":12021"
	tokenHeader = "x-chat-token"
)

// the server
type server struct {
	Host, Password string
	lock           sync.RWMutex
	chatclients    map[string]*Client
	chatgroups     map[string]*Group
	clienttoken    map[string]string
}

type Group struct {
	name      string
	ch        chan chat.Message
	clients   []string
	WaitGroup *sync.WaitGroup
}

type Client struct {
	name      string
	groups    []string
	ch        chan chat.Message
	WaitGroup *sync.WaitGroup
}

// AddClient adds a new client n to the server.

func (s *server) AddChatClient(n string) {

	s.lock.Lock()
	defer s.lock.Unlock()

	c := &Client{
		name:      n,
		ch:        make(chan chat.Message, 100),
		WaitGroup: &sync.WaitGroup{},
	}

	log.Print("added client " + n)
	s.chatclients[n] = c
}

//  add a new group to the server.

func (s *server) AddChatGroup(n string) {

	s.lock.Lock()
	defer s.lock.Unlock()

	g := &Group{
		name:      n,
		ch:        make(chan chat.Message, 100),
		WaitGroup: &sync.WaitGroup{},
	}

	log.Print("Added a chat group " + g.name)
	s.chatgroups[n] = g
	s.chatgroups[n].WaitGroup.Add(1)
}

// checks if a client name already exists on the server.

func (s *server) RegisteredClient(n string) bool {

	s.lock.RLock()
	defer s.lock.RUnlock()
	for c := range s.chatclients {
		if c == n {
			return true
		}
	}

	return false
}

//checks if a group name already exists on the server.

func (s *server) NotAvailableGroupName(groupName string) bool {

	s.lock.RLock()
	defer s.lock.RUnlock()
	for group := range s.chatgroups {
		if group == groupName {
			return true
		}
	}

	return false
}

//cheks if a given client joined a given group
func (s *server) ClientJoinedGroup(clientName string, groupName string) bool {

	for _, c := range s.chatgroups[groupName].clients {

		if clientName == c {
			return true
		}
	}
	return false
}

// remove a client form a given group
func (s *server) RemoveClientFromGroup(clientName string, groupName string) error {

	if s.ClientJoinedGroup(clientName, groupName) {
		// remove group name from client group
		c := s.chatclients[clientName].groups
		for i, g := range c {
			if groupName == g {
				c[i] = c[len(c)-1]
				c = c[:len(c)-1]
				s.chatclients[clientName].groups = c

				// remove the group if there is one client
				if len(s.chatgroups[g].clients) == 1 {
					delete(s.chatgroups, g)
				} else {

					//remove  client name  from the group client

					gc := s.chatgroups[g].clients
					for j, cl := range gc {
						if cl == clientName {

							gc[j] = gc[len(gc)-1]
							gc = gc[:len(gc)-1]
							s.chatgroups[g].clients = gc
						}

					}

				}

			}

		}
		return nil
	}
	return errors.New("can not found use with this name in this group ")
}

// remove client from any chat group

func (s *server) RemoveClient(clientName string) error {

	// TODO: There is some deadlock here when a user attempts to quit
	// 		 the chat app with !exit.

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.RegisteredClient(clientName) {

		delete(s.chatclients, clientName)

		for _, g := range s.chatclients[clientName].groups {

			s.RemoveClientFromGroup(clientName, g)
		}

	}
	return errors.New("Client (" + clientName + ") is not registred")
}

// add a client to a group.

func (s *server) AddClientToChatGroup(clientName string, groupName string) error {

	//lock.Lock()
	//defer lock.Unlock()

	s.chatgroups[groupName].WaitGroup.Add(1)
	defer s.chatgroups[groupName].WaitGroup.Done()
	// group did not exist
	if !s.NotAvailableGroupName(groupName) {
		return errors.New("Group (" + groupName + ") did not exist")
	}

	// client is not registered yet

	if !s.RegisteredClient(clientName) {
		return errors.New("Client(" + clientName + ") is not registered")
	}
	// client already joined group
	if s.ClientJoinedGroup(clientName, groupName) {
		return errors.New("Client(" + clientName + ") already joined group(" + groupName + ")")
	}
	s.chatgroups[groupName].clients = append(s.chatgroups[groupName].clients, clientName)
	s.chatclients[clientName].groups = append(s.chatclients[clientName].groups, groupName)

	log.Println("Added " + clientName + " to group" + groupName)
	return nil
}

// get all of the currently connected clients to the server.
func (s *server) GetChatClientList(ctx context.Context, in *chat.Empty) (*chat.ChatClientList, error) {

	var cl []string
	for key := range s.chatclients {
		cl = append(cl, key)
	}

	log.Print("this is the list of current clients ")
	log.Print(cl)

	return &chat.ChatClientList{Clients: cl}, nil
}

// It returns a list of  all chatgroups.
func (s *server) GetChatGroupList(ctx context.Context, in *chat.Empty) (*chat.ChatGroupList, error) {

	var grp []string
	for groupName := range s.chatgroups {
		grp = append(grp, groupName)
	}

	log.Print("this the list of current chat groups ")
	log.Print(grp)

	return &chat.ChatGroupList{Groups: grp}, nil
}

// It returns a list of clients of a group.
func (s *server) GetChatGroupClientList(ctx context.Context, in *chat.ChatGroup) (*chat.ChatClientList, error) {

	grpname := in.Name

	if !s.NotAvailableGroupName(grpname) {
		return &chat.ChatClientList{}, errors.New("that group doesn't exist")
	}

	list := s.chatgroups[grpname].clients

	log.Print("this is group " + grpname + " members list: ")
	log.Print(list)

	return &chat.ChatClientList{Clients: list}, nil
}

// Register will add the user to the server's collection of users (and by extension restrict the username).
// It returns an empty object and an error.
func (s *server) Register(ctx context.Context, in *chat.ChatClient) (*chat.Empty, error) {

	name := in.Sender
	if s.RegisteredClient(name) {
		return nil, errors.New("that client already registered")
	}

	s.AddChatClient(name)
	return &chat.Empty{}, nil
}

// removes a user from the server

func (s *server) UnRegister(ctx context.Context, in *chat.ChatClient) (*chat.Empty, error) {

	cl := in.Sender

	log.Print("Unregistering client " + cl)

	err := s.RemoveClient(cl)

	if err != nil {
		return nil, err
	}
	log.Print("Unregistered client " + cl)

	return &chat.Empty{}, nil
}

// creates a chat group if the name is availabe.
// It returns an empty object and an error.
func (s *server) CreateChatGroup(ctx context.Context, in *chat.ChatGroup) (*chat.Empty, error) {

	clName := in.Client
	grpName := in.Name

	log.Printf(clName + " is attempting creating " + grpName)

	if !s.NotAvailableGroupName(grpName) {
		s.AddChatGroup(grpName)
		return &chat.Empty{}, nil
	}

	return &chat.Empty{}, errors.New("the group name is not available")
}

// let a user to an existing group.

func (s *server) JoinChatGroup(ctx context.Context, in *chat.ChatGroup) (*chat.Empty, error) {

	clName := in.Client
	grpName := in.Name

	log.Printf(clName + " is trying to joing group: " + grpName)

	if s.NotAvailableGroupName(grpName) {
		s.AddClientToChatGroup(clName, grpName)

		return &chat.Empty{}, nil
	}

	return &chat.Empty{}, errors.New("a group with that name doesn't exist")
}

// LeaveRoom removes the user from their group.
// It returns an empty object and an error.
func (s *server) LeaveChatRoom(ctx context.Context, in *chat.ChatGroup) (*chat.Empty, error) {

	clName := in.Client
	grpName := in.Name

	if !s.NotAvailableGroupName(grpName) {
		return &chat.Empty{}, errors.New("group:" + grpName + " doesn't exist")

	} else if !s.RegisteredClient(clName) {
		return &chat.Empty{}, errors.New("the client name " + clName + " is not registered")
	} else {
		msg := chat.Message{Sender: clName, Receiver: grpName, Body: clName + " left chat!\n"}

		s.BroadcastMessage(grpName, msg)
		s.RemoveClientFromGroup(clName, grpName)
		return &chat.Empty{}, nil
	}
}

// Broadcast takes any messages that need to be sent and sorts them by group. It then
// adds  messages to message channel of each member of a group.

func (s *server) BroadcastMessage(grpName string, msg chat.Message) {

	s.lock.Lock()
	defer s.lock.Unlock()

	for grp := range s.chatgroups {
		log.Printf(grpName + ":")
		if grp == grpName {
			log.Printf(msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Body)
			for _, c := range s.chatgroups[grp].clients {

				if c == msg.Sender && msg.Body == msg.Sender+" left chat!\n" {

					s.chatclients[c].ch <- msg

				} else if c != msg.Sender {

					log.Printf(msg.Sender + "sending message to " + c + "...")
					s.chatclients[c].ch <- msg
				}
			}
		}
	}
}

// ListenToClient listens on the incoming stream for any messages. It adds those messages to the channel.
// It doesn't return anything.
func Listen(stream chat.ChatService_RouteChatServer, messages chan<- chat.Message) {

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
		}
		if err != nil {
		} else {
			log.Printf(msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Body)
			messages <- *msg
		}

	}

}

// RouteChat handles the routing of all messages on the stream.
// It returns an error.
func (s *server) RouteChat(stream chat.ChatService_RouteChatServer) error {

	msg, err := stream.Recv()

	if err != nil {
		return err
	}

	log.Printf(msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Body)

	outbox := make(chan chat.Message, 100)

	go Listen(stream, outbox)

	for {
		select {
		case outMsg := <-outbox:
			s.BroadcastMessage(msg.Receiver, outMsg)
		case inMsg := <-s.chatclients[msg.Sender].ch:
			log.Println("Sending message to channel: ")
			log.Println(s.chatclients[msg.Sender])
			log.Println("Sending message to STREAM: ")
			log.Println(stream)
			stream.Send(&inMsg)
		}
	}
}

func (s *server) genToken() string {
	tkn := make([]byte, 4)
	rand.Read(tkn)
	return fmt.Sprintf("%x", tkn)
}
func (s *server) Login(ctx context.Context, req *chat.ClientLoginRequest) (*chat.ClientLoginResponse, error) {
	switch {
	case req.Password != s.Password:
		return nil, status.Error(codes.Unauthenticated, "password is incorrect")
	case req.Name == "":
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	tkn := s.genToken()
	s.setName(tkn, req.Name)

	log.Println(tkn + "," + req.Name + "has logged in")

	return &chat.ClientLoginResponse{Token: tkn}, nil
}

//logout from server
func (s *server) Logout(ctx context.Context, req *chat.ClientLogoutRequest) (*chat.ClientLogoutResponse, error) {
	name, ok := s.deleteToken(req.Token)
	if !ok {
		return nil, status.Error(codes.NotFound, "token not found")
	}
	log.Printf(name + " logged out")
	return new(chat.ClientLogoutResponse), nil
}

func (s *server) getName(tkn string) (string, bool) {
	s.lock.RLock()
	name, ok := s.clienttoken[tkn]
	s.lock.RUnlock()
	return name, ok
}

func (s *server) setName(tkn string, name string) {
	s.lock.Lock()
	s.clienttoken[tkn] = name
	s.lock.Unlock()
}

func (s *server) deleteToken(tkn string) (name string, ok bool) {
	name, ok = s.getName(tkn)

	if ok {
		s.lock.Lock()
		delete(s.chatclients, tkn)
		s.lock.Unlock()
	}

	return name, ok
}

func (s *server) extractToken(ctx context.Context) (tkn string, ok bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md[tokenHeader]) == 0 {
		return "", false
	}

	return md[tokenHeader][0], true
}

func main() {

	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}
	log.Println("server listening on port" + port)
	// Initializes the gRPC server.
	s := grpc.NewServer()
	var mlock = &sync.RWMutex{}
	var clients = make(map[string]*Client)
	var groups = make(map[string]*Group)

	// Register the server with gRPC.
	chat.RegisterChatServiceServer(s, &server{lock: *mlock, chatclients: clients,
		chatgroups: groups, Host: port, Password: "nada"})

	// Register reflection service on gRPC server.
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
