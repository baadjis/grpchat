#grpchat
A client/ server (group or one to one) chat grpc app using golang

## how to 

### run server
 to start the server  run  ```go run server.go ```

### run client
 to start the client(s) we need to launch  two files by   ```go run client.go cmd.go ```
 and connect to the server by:
  * enter the sever ip:port ,by default use ```localhost:16180``` but you can change  it on the sever side.
  * enter your username for the session
  * finaly view the top menu to navigate (create group ,group options ,inbox options)

### command:
  * to disconect the server press ```cltr+c``` or type ```!exit``` 
  * type  ```!back``` to go back to the top menu 
  * type  ```!leave```  to leave chatroom