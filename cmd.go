package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/baadjis/grpchat/chat"
	"github.com/fatih/color"
)

// Stores the main color for all the entry dialogs.
const (
	promptColor = color.FgHiMagenta
)

// RandColor picks a random from a stored list of colors.
// It returns a single color attribute.
func RandColor() color.Attribute {

	c := []color.Attribute{color.FgHiCyan, color.FgHiGreen, color.FgHiRed, color.FgHiWhite, color.FgHiYellow, color.FgHiMagenta}
	return c[(len(c)*rand.Intn(20)+rand.Intn(10)+2)%len(c)]
}

// AddSpacing adds n-new lines to console.
// It doesn't return anything.
func AddSpacing(n int) {

	for i := 0; i < n; i++ {
		fmt.Println()
	}
}

// StartMessage displays the menu text for after a user connects to the server.
// It doesn't return anything.
func StartMessage() {

	AddSpacing(1)
	fmt.Println("Welcome to grpchat!")
	Frame()
	fmt.Println("In order to begin chatting, you must first chose a server and username. It cannot be one")
	fmt.Println("that is already in user on the server. Remember, your username only lasts for as long as")
	fmt.Println("you are logged into the server!")
	AddSpacing(1)
}

// WelcomeMessage displays a colored string welcoming the user to the server.
// It doesn't return anything.
func WelcomeMessage(c chat.ChatServiceClient, u string) {

	AddSpacing(1)
	u = "Welcome " + u + "!"
	for _, l := range u {
		color.New(RandColor()).Print(string(l))
	}
	n, _ := c.GetChatClientList(context.Background(), &chat.Empty{})
	g, _ := c.GetChatGroupList(context.Background(), &chat.Empty{})

	fmt.Print(" There are currently " + strconv.Itoa(len(n.Clients)) + " member(s) logged in and " + strconv.Itoa(len(g.Groups)) + " group(s).")
	AddSpacing(1)
}

// TopMenuText displays the option text for the main menu.
// It doesn't return anything.
func TopMenuText() {

	fmt.Println("Main Menu")
	AddSpacing(1)
	fmt.Println("1) Create a Group")
	fmt.Println("2) View Group Options")
	fmt.Println("3) view inbox options")
	fmt.Println("4) exit chat ")
	AddSpacing(1)
	color.New(promptColor).Print("Main> ")
}

// GroupMenuText displays the option text for the group menu.
// It doesn't return anything.
func GroupMenuText() {

	fmt.Println("View Groups Menu")
	AddSpacing(1)
	fmt.Println("Below is a list of menu options for groups.")
	AddSpacing(1)
	fmt.Println("1) View a Group's Members")
	fmt.Println("2) Refresh List of Groups")
	fmt.Println("3) Join a Group")
	fmt.Println("4) Go back")
	AddSpacing(1)
	color.New(promptColor).Print("Groups> ")
}
// inbox menu text
func InboxMenuText() {

	fmt.Println("Inbox Menu")
	AddSpacing(1)
	fmt.Println("1) view inbox invitation")
	fmt.Println("2) send invitation to someone")
	fmt.Println("3) accept or reject invitation from someone")
	fmt.Println("4) exit inbox")

	AddSpacing(1)
	color.New(promptColor).Print("Inbox> ")
}

// ViewGroupMemMenuText displays option text to view a group.
// It doesn't return anything.
func ViewGroupMemMenuText() {

	AddSpacing(1)
	fmt.Println("Enter the group name that you would like to view! Enter !back to go back to the menu.")
	AddSpacing(1)
}

// Frame gives some nice formatting structure to the output.
func Frame() {

	fmt.Println("------------------------------------------")
}

// SetServer handles the input for the chat server address.
// It returns a string which contains the ip:port of the chat server.
func SetServer(r *bufio.Reader) string {

	StartMessage()

	fmt.Print("Please specify the server IP: ")
	t, _ := r.ReadString('\n')
	t = strings.TrimSpace(t)
	s := strings.Split(t, ":")
	ip := s[0]
	p := s[1]
	address := ip + ":" + p

	return address
}

// SetName sets the username for the user.
// It returns a string containing the username of the client.
func SetName(c chat.ChatServiceClient, r *bufio.Reader) string {
	for {
		fmt.Printf("Enter your username: ")
		n, err := r.ReadString('\n')
		if err != nil {
			fmt.Print(err)
		} else {
			uName := strings.TrimSpace(n)
			if len(uName) < 3 {
				AddSpacing(1)
				color.New(color.FgHiRed).Println("Your username must be at least 3 characters long.")
			} else {
				_, err = c.Register(context.Background(), &chat.ChatClient{Sender: uName})

				if err != nil {
					AddSpacing(1)
					color.New(color.FgHiRed).Println("That username already exists. Please choose a new one! ")
				} else {
					WelcomeMessage(c, uName)
					return uName
				}
			}
		}
	}
}

// CreateGroup handles the create group menu option.
// It returns a string which contains the keyword !back allowing it to escape the input as well as an error.
func CreateChatGroup(c chat.ChatServiceClient, r *bufio.Reader, uName string) (string, error) {

	for {
		AddSpacing(1)
		fmt.Println("Enter the name of the group or type !back to go back to the main menu.")
		color.New(promptColor).Print("Join> ")
		g, err := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if err != nil {
			return "", err
		} else if g != "!back" {
			_, nerr := c.CreateChatGroup(context.Background(), &chat.ChatGroup{Client: uName, Name: g})

			if nerr != nil {
				AddSpacing(1)
				color.New(color.FgRed).Println("The group name \"" + g + "\" has already been chosen. Please select a new one.")
			} else {
				c.JoinChatGroup(context.Background(), &chat.ChatGroup{Client: uName, Name: g})
				AddSpacing(1)
				color.New(color.FgGreen).Println("Created and joined group named " + g)
				return g, nil
			}
		} else {
			return g, nil
		}
	}
}

// handles the join group menu option.

func JoinChatGroup(c chat.ChatServiceClient, r *bufio.Reader, u string) string {

	for {
		fmt.Println("Enter the name of the group as it appears in the group list or enter !back to go back to the Group menu.")
		color.New(promptColor).Print("Group Name> ")
		g, _ := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if g == "!back" {
			return g
		}

		_, err := c.JoinChatGroup(context.Background(), &chat.ChatGroup{Client: u, Name: g})

		if err != nil {
			AddSpacing(1)
			color.New(color.FgRed).Println("The group name \"" + g + "\" doesn't exist. Please check again.")
			AddSpacing(1)
		} else {
			color.New(color.FgGreen).Println("Joined " + g)
			return g
		}
	}
}
// get chat groups or  invitattions for inbox chat list
func GetChatGroupsOrInvitation(c chat.ChatServiceClient)([]string,[]string){
	t, _ := c.GetChatGroupList(context.Background(), &chat.Empty{})	
	l :=t.Groups
	groups :=make([]string, 0)
	invitations :=make([]string, 0)
	for _,g :=range l{
		if strings.Contains(g,"+"){
			invitations = append(invitations,strings.Split(g,"+")[0])
		}else{
			groups = append(groups,g)
		}

	}
   return groups,invitations
}
// ListGroups handles listing all of the groups stored on the server.
// It doesn't return anything.
func ListChatGroups(c chat.ChatServiceClient, r *bufio.Reader) {

	l, _ := GetChatGroupsOrInvitation(c)
	

	if len(l) == 0 {
		AddSpacing(1)
		color.New(color.FgYellow).Println("There are no groups created yet!")
	} else {
		AddSpacing(1)
		fmt.Println("Current groups able to join:")
		for i, g := range l {
			fmt.Println("  " + strconv.Itoa(i+1) + ") " + g)
		}
	}

}

// ListGroupMembers handles listing the members of a specific group.
// It returns an error.
func ListChatGroupMembers(c chat.ChatServiceClient, r *bufio.Reader, u string) error {

	for {
		color.New(promptColor).Print("View> ")
		t, _ := c.GetChatGroupList(context.Background(), &chat.Empty{})
		n := len(t.Groups)

		if n == 0 {
			AddSpacing(2)
			color.New(color.FgYellow).Println("There are currently no groups created!")
			return nil
		}

		g, err := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if err != nil {
			return err
		} else if g == "!back" {
			return nil
		} else {
			ls, err := c.GetChatGroupClientList(context.Background(), &chat.ChatGroup{Client: u, Name: g})
			if err != nil {
				color.New(color.FgRed).Println("Please double check that the group name you entered actually exists.")
			} else {
				fmt.Println("Members of " + g)
				for i, c := range ls.Clients {
					fmt.Println("  " + strconv.Itoa(i+1) + ") " + c)
				}

				return nil
			}
		}
	}
}

func IsRegistered(c chat.ChatServiceClient, uName string) bool {
	clientList, _ := c.GetChatClientList(context.Background(), &chat.Empty{})
	for _, cl := range clientList.Clients {
		if cl == uName {
			return true
		}
	}
	return false
}
func JoinedGroup(c chat.ChatServiceClient, uName string, gName string) bool {
	members, _ := c.GetChatGroupClientList(context.Background(), &chat.ChatGroup{Name: gName})
	for _, cl := range members.Clients {
		if cl == uName {
			return true
		}
	}

	return false
}

// invite someone for inboxchat

func InboxInvitation(c chat.ChatServiceClient, r *bufio.Reader, uName string) (string, error) {

	for {
		AddSpacing(1)
		fmt.Println("Enter the name of person you want to invite for inbox chat.")
		color.New(promptColor).Print("Invite> ")
		other, err := r.ReadString('\n')
		other = strings.TrimSpace(other)
		g := uName + "+" + other
		if err != nil {
			return "", err
		} else if other != "!back" && IsRegistered(c, other) {

			_, nerr := c.CreateChatGroup(context.Background(), &chat.ChatGroup{Client: uName, Name: g})

			if nerr != nil {
				AddSpacing(1)
				color.New(color.FgRed).Println("invitation already sent")
			} else {
				c.JoinChatGroup(context.Background(), &chat.ChatGroup{Client: uName, Name: g})
				AddSpacing(1)
				color.New(color.FgGreen).Println("sent inbox invitation to: " + other)
				return g, nil
			}
		} else {
			return g, nil
		}
	}
}



// list chat invitation for current user
func ListInvitations(c chat.ChatServiceClient, uName string) {
	fmt.Println("invitations:")

	_,list := GetChatGroupsOrInvitation(c)
	if len(list) > 0 {
		for i, inv := range list {

			Frame()
			println(strconv.Itoa(i+1) + " )invitation from :" + inv)

		}
	} else {
		println("you have no invitation")
	}
}
func CheckInvitation(c chat.ChatServiceClient, u string, other string) bool {
	_,list := GetChatGroupsOrInvitation(c)

	for _, inv := range list {
		if inv == other {

			return true
		}
	}

	fmt.Println("you have no invitation from:" + other + "!!")

	return false
}

// accept invitation from someone
func AcceptOrRejectInvitation(c chat.ChatServiceClient, r *bufio.Reader, u string) string{
	_,list := GetChatGroupsOrInvitation(c)

	if len(list) > 0 {
		
			fmt.Println("type the name of someone to accept or reject invitation")
			other, _ := r.ReadString('\n')
			other = strings.TrimSpace(other)
			g := other +"+"+ u
			if CheckInvitation(c, u, other) {
				println(">Accept " + other + " y(yes) or n(no): ")
				i, _ := r.ReadString('\n')
				i = strings.TrimSpace(i)
				switch answer := i; answer {
				case "y": //accept
					_,err := c.JoinChatGroup(context.Background(), &chat.ChatGroup{Client: u, Name: g})
					if err ==nil {
						color.New(color.FgGreen).Println("Joined " + other)
						
						return g
					}
					

				case "n":
					c.LeaveChatRoom(context.Background(), &chat.ChatGroup{Client: u, Name: g})
					return "!back"
				default:
					fmt.Println("please answer y(yes) or n(no)")
				}

			}
		
	} else {
		fmt.Println("you have no invitation")
	}
 return "!back"
}

//
// TopMenu handles displaying the menu to the client.
// It returns the group name for the user and an error.
func TopMenu(c chat.ChatServiceClient, r *bufio.Reader, u string) (string, error) {
	//func TopMenu(c pb.ChatClient, u string) (string, error) {
	log.Println("In TopMenu")

	//r := bufio.NewReader(os.Stdin)

	for {
		Frame()
		TopMenuText()
		i, _ := r.ReadString('\n')
		i = strings.TrimSpace(i)

		switch input := i; input {
		case "1": // Create group
			g, err := CreateChatGroup(c, r, u)

			if err != nil {
				return g, err
			} else if g != "!back" {
				return g, nil
			}
		case "2": // View Group Menu
			g, err := DisplayGroupMenu(c, r, u)

			if err != nil {
				return g, err
			} else if g != "!back" {
				return g, nil
			}
		case "3": // inbox menu
		 return DisplayInboxMenu(c,r,u)
		    

		case "4": // exit client
		    c.UnRegister(context.Background(), &chat.ChatClient{Sender: u})
		    os.Exit(0)
		
		default: // Error
			color.New(color.FgRed).Println("Please enter a valid selection between 1 and 3.")
		}
	}
}

// displays the menu for the group options.

func DisplayGroupMenu(c chat.ChatServiceClient, r *bufio.Reader, u string) (string, error) {

	ListChatGroups(c, r)

	for {
		Frame()
		GroupMenuText()
		i, _ := r.ReadString('\n')
		i = strings.TrimSpace(i)

		switch input := i; input {
		case "1": // View Group Members
			ViewGroupMemMenuText()
			err := ListChatGroupMembers(c, r, u)
			if err != nil {
				return "", err
			}
		case "2": // Refresh Group List
			ListChatGroups(c, r)
			break
		case "3": // Join Group
			g := JoinChatGroup(c, r, u)
			if g != "!back" {
				return g, nil
			}
		case "4": // Go Back
			return "!back", nil
		default: // Error
			color.New(color.FgRed).Println("Please enter a valid selection between 1 and 4.")
		}
	}
}
func DisplayInboxMenu(c chat.ChatServiceClient, r *bufio.Reader, u string) (string, error){
	log.Println("In TopMenu")
	for {
		Frame()
		InboxMenuText()
		i, _ := r.ReadString('\n')
		i = strings.TrimSpace(i)

		switch input := i; input {

		case "1": // list invitations
			ListInvitations(c, u)
		case "2": // send  invitation to someone
			g,_:=InboxInvitation(c, r, u)
			return g,nil
		case "3":
			g:=AcceptOrRejectInvitation(c, r, u)
			return g,nil
		default: // Error
			color.New(color.FgRed).Println("Please enter a valid selection between 1 and 3.")
		}
	}
}