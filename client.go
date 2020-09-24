package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/baadjis/grpchat/chat"
	"github.com/fatih/color"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Listener struct {
	MessageChanel    chan chat.Message
	MessageWaitGroup *sync.WaitGroup
	chattingState    bool
	stream           chat.ChatService_RouteChatClient
	SignalWaitGroup  *sync.WaitGroup
	SignalChanel     chan os.Signal
}

func NewListener() *Listener {
	listener := &Listener{
		chattingState:    false,
		MessageChanel:    make(chan chat.Message),
		stream:           nil,
		MessageWaitGroup: &sync.WaitGroup{},
		SignalChanel:     make(chan os.Signal),
		SignalWaitGroup:  &sync.WaitGroup{},
	}
	listener.MessageWaitGroup.Add(1)
	listener.SignalWaitGroup.Add(1)
	return listener
}

func (s *Listener) StopListeningMessage() {
	log.Print("[Stop]: Entered Stop.")
	log.Print(s.SignalChanel)
	close(s.SignalChanel)
	log.Print("[Stop]: Waiting.")
	s.MessageWaitGroup.Wait()
	log.Print("[Stop]: Done.")
}

// ControlExit handles any interrupts during program execution.
// Note: The routine control is dictated by the existence of a stream. If one is present, the user is in a group and needs
// to be removed. Otherwise, the user is still in the menu system.
// It doesn't return anything.
func (s *Listener) ControlExit(c chat.ChatServiceClient, u string, g string) {

	log.Print("[ControlExit]: Entered.")

	defer s.SignalWaitGroup.Done()
	signal.Notify(s.SignalChanel, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-s.SignalChanel:

			if s.chattingState {
				log.Print("I am still chatting.")
				s.stream.Send(&chat.Message{Sender: u, Receiver: g, Body: u + " left chat!\n"})
				ExitClient(c, u, g)
				return
			}

			log.Print("I am no longer chatting.")
			ExitClient(c, u, g)
			os.Exit(1)
			return
		}
	}
}

// ExitClient handles removing the client from the server and exiting the program.
// It doesn't return anything.
func ExitClient(c chat.ChatServiceClient, u string, g string) {

	c.UnRegister(context.Background(), &chat.ChatClient{Sender: u})
	os.Exit(1)
}

// ListenToClient listens to the client for input and adds that input to the sQueue with
// the username of the sender, group name, and the message.
// It doesn't return anything.
func ListenToClient(messagesQueue *Listener, reader *bufio.Reader, uName string, gName string) {

	log.Println("Start listening.")
	defer messagesQueue.MessageWaitGroup.Done()

	for {
		msg, _ := reader.ReadString('\n')
		if strings.TrimSpace(msg) == "!leave" {
			log.Println("Stop listening chat.")
			messagesQueue.MessageChanel <- chat.Message{Sender: uName, Body: msg, Receiver: gName}
			return
		}
		log.Println("Adding message to the queue.")
		messagesQueue.MessageChanel <- chat.Message{Sender: uName, Body: msg, Receiver: gName}
	}
}

// ReceiveMessages listens on the client's (NOT the client's group) stream and adds any incoming
// message to the client's inbox.
// It doesn't return anything.
func ReceiveMessages(inbox *Listener, stream chat.ChatService_RouteChatClient, u string) {

	log.Println("[ReceiveMessages]: Starting.")
	defer inbox.MessageWaitGroup.Done()

	for {
		log.Println("listening for new messages.")
		msg, _ := stream.Recv()
		log.Println("Received message: " + msg.Body)
		if msg.Body == u+" left chat!\n" {
			log.Println("Found special signal!")
			return
		}

		inbox.MessageChanel <- *msg
	}
}

// DisplayCurrentMembers displays the members who are currently in the group chat.
// It doesn't return anything.
func CurrentMembers(c chat.ChatServiceClient, g string) {

	m, _ := c.GetChatGroupClientList(context.Background(), &chat.ChatGroup{Name: g})
	if len(m.Clients) > 0 {
		fmt.Print("Current Members: ")
		for i := 0; i < len(m.Clients); i++ {
			if i == len(m.Clients)-1 {
				fmt.Print(m.Clients[i])
			} else {
				fmt.Print(m.Clients[i] + ", ")
			}
		}

	}
}

//initialise message listener and start chatting

func StartChat(stream chat.ChatService_RouteChatClient, c chat.ChatServiceClient, listener *Listener, r *bufio.Reader, u string, g string) *Listener {

	//t, cancel := context.WithCancel(ctx)
	//stream, serr := c.RouteChat(t)
	//defer cancel()

	CurrentMembers(c, g)

	//if serr != nil {
	//	fmt.Print(serr)
	//} else {
	listenerQueue := NewListener() // Creates the sQueue with a channel and waitgroup.

	go ListenToClient(listenerQueue, r, u, g)
	go ReceiveMessages(listenerQueue, stream, u)

	// TODO: Find out why the first message is always dropped so an empty message needn't be sent.
	stream.Send(&chat.Message{Sender: u, Receiver: g, Body: ""})
	stream.Send(&chat.Message{Sender: u, Receiver: g, Body: "joined chat!\n"})
	listener.chattingState = true
	listener.stream = stream
	return listenerQueue

}

func Chat(conn *grpc.ClientConn, stream chat.ChatService_RouteChatClient, c chat.ChatServiceClient, listener *Listener, r *bufio.Reader, u string, g string) bool {

	listenerQueue := StartChat(stream, c, listener, r, u, g)
	AddSpacing(1)
	fmt.Println("good chat with " + g + ".")
	Frame()

	for {
		select {
		case toSend := <-listenerQueue.MessageChanel:
			switch msg := strings.TrimSpace(toSend.Body); msg {
			case "!members":
				log.Println("!members.")
				CurrentMembers(c, g)
			case "!leave":
				log.Println("!leave.")
				c.LeaveChatRoom(context.Background(), &chat.ChatGroup{Client: u, Name: g})
				listenerQueue.StopListeningMessage()

				//stream.CloseSend()
				log.Println("HEY LOOK")
				//cancel()
				log.Println(context.Canceled)
				return true
			case "!exit":
				log.Println("!exit.")
				stream.Send(&chat.Message{Sender: u, Receiver: g, Body: u + " left chat!\n"})
				ExitClient(c, u, g)
				//stream.CloseSend()
				//cancel()
				conn.Close()
				return false
			case "!help":
				log.Println("[Main]: I'm in !help.")
				AddSpacing(1)
				fmt.Println("The following commands are available to you: ")
				color.New(color.FgHiYellow).Print("   !members")
				fmt.Print(": Lists the current members in the group.")

				AddSpacing(1)
				color.New(color.FgHiYellow).Print("   !exit")
				fmt.Println(": Leaves the chat server.")
				AddSpacing(1)

			default:
				log.Println("[Main]: Sending the message.")
				stream.Send(&toSend)
			}
		case received := <-listenerQueue.MessageChanel:
			log.Println("[Receiving the message.")
			if received.Body != "!leave" {
				fmt.Printf("%s> %s", received.Sender, received.Body)
			}
		}
	}

}
func main() {

	r := bufio.NewReader(os.Stdin)

	var uName string // Client username
	var gName string // Client's chat group

	a := SetServer(r)

	// Set up a connection to the server.
	conn, err := grpc.Dial(a, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	} else {
		fmt.Printf("\nYou have successfully connected to %s! To disconnect, hit ctrl+c or type !exit.\n\n", a)
	}

	// Close the connection after main returns.
	defer conn.Close()

	// Create the client
	c := chat.NewChatServiceClient(conn)
	ctx := context.Background()
	stream, serr := c.RouteChat(ctx)
	if serr != nil {
		log.Fatal(serr)
	}

	uName = SetName(c, r)
	showMenu := true // Control whether the user sees the menu or exits.
	m := NewListener()
	go m.ControlExit(c, uName, gName)

	for showMenu {
		gName, err = TopMenu(c, r, uName)

		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		log.Print("Showing context")
		log.Print(ctx)
		showMenu = Chat(conn, stream, c, m, r, uName, gName)
	}
}
