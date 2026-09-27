package api

import (
	"net/http"
)

func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.GET("/", rt.getHelloWorld)
	rt.router.GET("/context", rt.wrap(rt.getContextReply))

	// doLogin
	rt.router.POST("/session", rt.doLogin)

	// setMyPhoto
	rt.router.PUT("/users/me/photo", rt.setMyPhoto)

	// getMyConversations
	rt.router.GET("/conversations", rt.getMyConversations)

	// getConversation
	rt.router.GET("/conversations/:conversationId", rt.getConversation)

	// createConversation
	rt.router.POST("/conversations", rt.createConversation)

	// createGroup
	rt.router.POST("/groups", rt.createGroup)

	// setGroupName
	rt.router.PUT("/groups/:groupId/name", rt.setGroupName)

	// setGroupPhoto
	rt.router.PUT("/groups/:groupId/photo", rt.setGroupPhoto)

	// addToGroup
	rt.router.POST("/groups/:groupId/members", rt.addToGroup)

	// leaveGroup
	rt.router.DELETE("/groups/:groupId/members/me", rt.leaveGroup)

	// searchUsers
	rt.router.GET("/users", rt.searchUsers)

	// sendMessage
	rt.router.POST("/conversations/:conversationId/messages", rt.sendMessage)

	// deleteMessage
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId", rt.deleteMessage)

	// commentMessage
	rt.router.PUT("/conversations/:conversationId/messages/:messageId/reactions", rt.commentMessage)

	// uncommentMessage
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId/reactions", rt.uncommentMessage)

	// forwardMessage
	rt.router.POST("/conversations/:conversationId/messages/:messageId/forward", rt.forwardMessage)

	// markMessagesRead
	rt.router.PUT("/conversations/:conversationId/read", rt.markMessagesRead)

	// Special routes
	rt.router.GET("/liveness", rt.liveness)

	rt.router.ServeFiles("/photos/*filepath", http.Dir("/tmp/photos"))

	return rt.router
}
