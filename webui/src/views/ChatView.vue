<script>
import GroupSettings from '../components/GroupSettings.vue';
import UserSettings from '../components/UserSettings.vue';
import CreateGroup from '../components/CreateGroup.vue';
import SearchUsers from '../components/SearchUsers.vue';

export default {
	data() {
		return {
			// Conversation list
			conversations: [],
			selectedConversationId: null,
			errormsg: null,
			loading: false,
			pollingInterval: null,
			chatPollingInterval: null,

			showSearch: false,

			// Pending new chat (user selected from search, chat not yet created)
			pendingUsername: null,

			showSettings: false,

			// Current conversation
			currentConversation: null,
			conversationLoading: false,
			conversationError: null,

			// Message input
			messageText: '',
			messageImageFile: null,
			sendingMessage: false,

			// Reply
			replyToMessage: null,

			// Reactions
			showEmojiPicker: null, // message ID of the message whose picker is open

			// Forward
			showForwardPicker: null, // message ID whose forward picker is open
			forwardError: null,

			showCreateGroup: false,

			// Group settings panel visibility
			showGroupSettings: false,
		}
	},
	computed: {
		currentUsername() {
			return localStorage.getItem('username') || '';
		},
		// Messages in chronological order (API returns reverse chronological)
		sortedMessages() {
			if (!this.currentConversation || !this.currentConversation.messages) return [];
			return [...this.currentConversation.messages].reverse();
		},
		selectedConversation() {
			return this.conversations.find(c => c.id === this.selectedConversationId) || null;
		},
		isGroupChat() {
			return this.currentConversation && this.currentConversation.type === 'group';
		},
		authToken() {
			return localStorage.getItem('token') || '';
		},
	},
	components: { GroupSettings, UserSettings, CreateGroup, SearchUsers },
	methods: {
		authHeader() {
			return { Authorization: 'Bearer ' + localStorage.getItem('token') };
		},

		// ── Conversation list ──────────────────────────────────────────────
		async loadConversations() {
			try {
				const response = await this.$axios.get('/conversations', {
					headers: this.authHeader(),
				});
				this.conversations = response.data || [];
			} catch (e) {
				if (e.response && e.response.status === 401) {
					this.$router.push('/login');
				}
			}
		},

		async deleteMessage(conversationId, messageId) {
		    try {
		        await this.$axios.delete(
		            '/conversations/' + conversationId + '/messages/' + messageId,
		            { headers: this.authHeader() }
		        );
		        await this.loadConversation(conversationId);
		    } catch (e) {
		        this.conversationError = e.toString();
		    }
		},

		async selectConversation(id) {
			this.selectedConversationId = id;
			this.pendingUsername = null;
			this.replyToMessage = null;
			this.showSearch = false;
			await this.loadConversation(id);
			await this.markRead(id);
			// Restart chat polling for this conversation
			clearInterval(this.chatPollingInterval);
			this.chatPollingInterval = setInterval(() => this.loadConversation(id), 4000);
		},

		async loadConversation(id) {
			this.conversationError = null;
			this.conversationLoading = true;
			try {
				const response = await this.$axios.get('/conversations/' + id, {
					headers: this.authHeader(),
				});
				this.currentConversation = response.data;
			} catch (e) {
				this.conversationError = e.toString();
			}
			this.conversationLoading = false;
		},

		async markRead(id) {
			try {
				await this.$axios.put('/conversations/' + id + '/read', null, {
					headers: this.authHeader(),
				});
				// Refresh conversation list to reset unread badge
				await this.loadConversations();
			} catch (e) {
				// Non-critical — ignore
			}
		},

		// ── Search & new conversation ──────────────────────────────────────
		selectUserFromSearch(username) {
			// Check if a conversation with this user already exists
			const existing = this.conversations.find(c => c.type === 'direct' && c.name === username);
			if (existing) {
				this.selectConversation(existing.id);
			} else {
				// Open a "pending" chat panel — conversation created on first message send
				this.pendingUsername = username;
				this.selectedConversationId = null;
				this.currentConversation = null;
				this.replyToMessage = null;
				clearInterval(this.chatPollingInterval);
			}
			this.showSearch = false;
		},

		// ── Send message ───────────────────────────────────────────────────
		async sendMessage() {
			const text = this.messageText.trim();
			const hasImage = !!this.messageImageFile;
			if (!text && !hasImage) return;

			this.sendingMessage = true;
			try {
				if (this.pendingUsername) {
					// First message — creates the conversation
					await this.$axios.post('/conversations', {
						targetUsername: this.pendingUsername,
						content: text || 'Photo',
					}, { headers: this.authHeader() });
					await this.loadConversations();
					// Find the newly created conversation and open it
					const created = this.conversations.find(c => c.type === 'direct' && c.name === this.pendingUsername);
					if (created) {
						this.pendingUsername = null;
						await this.selectConversation(created.id);
					}
				} else if (this.selectedConversationId) {
					if (hasImage) {
						// Image message
						const formData = new FormData();
						formData.append('photo', this.messageImageFile);
						await this.$axios.post(
							'/conversations/' + this.selectedConversationId + '/messages',
							formData,
							{ headers: { ...this.authHeader(), 'Content-Type': 'multipart/form-data' } }
						);
						this.messageImageFile = null;
					} else {
						// Text message
						const body = { content: text };
						if (this.replyToMessage) {
							body.replyToMessageId = this.replyToMessage.id;
						}
						await this.$axios.post(
							'/conversations/' + this.selectedConversationId + '/messages',
							body,
							{ headers: this.authHeader() }
						);
					}
					this.replyToMessage = null;
					this.messageText = '';
					await this.loadConversation(this.selectedConversationId);
					await this.loadConversations();
				}
			} catch (e) {
				this.conversationError = e.toString();
			}
			this.messageText = '';
			this.sendingMessage = false;
		},

		onImageSelected(event) {
			this.messageImageFile = event.target.files[0] || null;
		},

		setReply(message) {
			this.replyToMessage = message;
		},

		cancelReply() {
			this.replyToMessage = null;
		},

		// ── Formatting helpers ─────────────────────────────────────────────
		formatTime(timestamp) {
			if (!timestamp) return '';
			const date = new Date(timestamp);
			const now = new Date();
			const isToday = date.toDateString() === now.toDateString();
			if (isToday) {
				return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
			}
			return date.toLocaleDateString([], { day: '2-digit', month: '2-digit', year: '2-digit' });
		},

		checkmarkIcon(status) {
			if (status === 'read') return '✓✓';
			if (status === 'received') return '✓';
			return '';
		},

		photoSrc(url) {
			if (!url) return null;
			if (url.startsWith('http')) return url;
			return __API_URL__ + url;
		},

		// ── Reactions ──────────────────────────────────────────────────────
		toggleEmojiPicker(messageId) {
			this.showEmojiPicker = this.showEmojiPicker === messageId ? null : messageId;
		},

		async addReaction(messageId, emoticon) {
			this.showEmojiPicker = null;
			try {
				await this.$axios.put(
					'/conversations/' + this.selectedConversationId + '/messages/' + messageId + '/reactions',
					{ emoticon: emoticon },
					{ headers: this.authHeader() }
				);
				await this.loadConversation(this.selectedConversationId);
			} catch (e) {
				this.conversationError = e.toString();
			}
		},

		async removeReaction(messageId) {
			try {
				await this.$axios.delete(
					'/conversations/' + this.selectedConversationId + '/messages/' + messageId + '/reactions',
					{ headers: this.authHeader() }
				);
				await this.loadConversation(this.selectedConversationId);
			} catch (e) {
				// 404 means no reaction existed — ignore silently
			}
		},

		myReaction(msg) {
			// Returns the current user's reaction on this message, or null
			if (!msg.reactions) return null;
			const r = msg.reactions.find(r => r.username === this.currentUsername);
			return r ? r.emoticon : null;
		},

		// ── Forward message ────────────────────────────────────────────────
		toggleForwardPicker(messageId) {
			this.showForwardPicker = this.showForwardPicker === messageId ? null : messageId;
			this.forwardError = null;
		},

		async forwardMessage(messageId, targetConversationId) {
			this.showForwardPicker = null;
			this.forwardError = null;
			try {
				await this.$axios.post(
					'/conversations/' + this.selectedConversationId + '/messages/' + messageId + '/forward',
					{ targetConversationId: targetConversationId },
					{ headers: this.authHeader() }
				);
				// If forwarding to the current conversation, refresh messages
				if (targetConversationId === this.selectedConversationId) {
					await this.loadConversation(this.selectedConversationId);
				}
				await this.loadConversations();
			} catch (e) {
				this.forwardError = e.toString();
			}
		},

		// ── GroupSettings event handlers ───────────────────────────────────
		async onGroupUpdated() {
			await this.loadConversations();
			await this.loadConversation(this.selectedConversationId);
		},

		// ── Component event handlers ────────────────────────────────────────
		onUserLogout() {
			this.$router.push('/login');
		},

		onUsernameChanged() {
			this.loadConversations();
		},

		async onGroupCreated(groupId) {
			this.showCreateGroup = false;
			await this.loadConversations();
			const created = this.conversations.find(c => c.id === groupId);
			if (created) await this.selectConversation(created.id);
		},

		onUserSelected(username) {
			this.showSearch = false;
			this.selectUserFromSearch(username);
		},

		onGroupLeft() {
			this.selectedConversationId = null;
			this.currentConversation = null;
			this.showGroupSettings = false;
			clearInterval(this.chatPollingInterval);
			this.loadConversations();
		},

	},

	mounted() {
		this.loadConversations();
		this.pollingInterval = setInterval(this.loadConversations, 4000);
	},

	beforeUnmount() {
		clearInterval(this.pollingInterval);
		clearInterval(this.chatPollingInterval);
	},
}
</script>

<template>
	<div class="d-flex" style="height: 100vh; overflow: hidden;">

		<!-- ── LEFT PANEL ─────────────────────────────────────────────────── -->
		<div class="d-flex flex-column border-end" style="width: 340px; min-width: 340px; background: #f5f5f5;">

			<!-- Header -->
			<div class="d-flex align-items-center justify-content-between px-3 py-2 border-bottom bg-white">
				<span class="fw-bold fs-5">WASAText</span>
				<div class="d-flex gap-2">
					<button class="btn btn-sm btn-outline-secondary" title="New group" @click="showCreateGroup = !showCreateGroup; showSearch = false; showSettings = false">👥</button>
					<button class="btn btn-sm btn-outline-secondary" title="New conversation" @click="showSearch = !showSearch; showSettings = false; showCreateGroup = false">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
							<path d="M11.742 10.344a6.5 6.5 0 1 0-1.397 1.398l3.85 3.85a1 1 0 0 0 1.415-1.415l-3.868-3.833zm-5.242 1.156a5.5 5.5 0 1 1 0-11 5.5 5.5 0 0 1 0 11z"/>
						</svg>
					</button>
					<button class="btn btn-sm btn-outline-secondary" title="Settings" @click="showSettings = !showSettings; showSearch = false; showCreateGroup = false">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
							<path d="M8 4.754a3.246 3.246 0 1 0 0 6.492 3.246 3.246 0 0 0 0-6.492zM5.754 8a2.246 2.246 0 1 1 4.492 0 2.246 2.246 0 0 1-4.492 0z"/>
							<path d="M9.796 1.343c-.527-1.79-3.065-1.79-3.592 0l-.094.319a.873.873 0 0 1-1.255.52l-.292-.16c-1.64-.892-3.433.902-2.54 2.541l.159.292a.873.873 0 0 1-.52 1.255l-.319.094c-1.79.527-1.79 3.065 0 3.592l.319.094a.873.873 0 0 1 .52 1.255l-.16.292c-.892 1.64.901 3.434 2.541 2.54l.292-.159a.873.873 0 0 1 1.255.52l.094.319c.527 1.79 3.065 1.79 3.592 0l.094-.319a.873.873 0 0 1 1.255-.52l.292.16c1.64.892 3.433-.902 2.54-2.541l-.159-.292a.873.873 0 0 1 .52-1.255l.319-.094c1.79-.527 1.79-3.065 0-3.592l-.319-.094a.873.873 0 0 1-.52-1.255l.16-.292c.892-1.64-.901-3.433-2.541-2.54l-.292.159a.873.873 0 0 1-1.255-.52l-.094-.319zm-2.633.283c.246-.835 1.428-.835 1.674 0l.094.319a1.873 1.873 0 0 0 2.693 1.115l.291-.16c.764-.415 1.6.42 1.184 1.185l-.159.292a1.873 1.873 0 0 0 1.116 2.692l.318.094c.835.246.835 1.428 0 1.674l-.319.094a1.873 1.873 0 0 0-1.115 2.693l.16.291c.415.764-.42 1.6-1.185 1.184l-.291-.159a1.873 1.873 0 0 0-2.693 1.116l-.094.318c-.246.835-1.428.835-1.674 0l-.094-.319a1.873 1.873 0 0 0-2.692-1.115l-.292.16c-.764.415-1.6-.42-1.184-1.185l.159-.291A1.873 1.873 0 0 0 1.945 8.93l-.319-.094c-.835-.246-.835-1.428 0-1.674l.319-.094A1.873 1.873 0 0 0 3.06 4.474l-.16-.292c-.415-.764.42-1.6 1.185-1.184l.292.159a1.873 1.873 0 0 0 2.692-1.115l.094-.319z"/>
						</svg>
					</button>
				</div>
			</div>

			<!-- Search panel -->
			<SearchUsers
				v-if="showSearch"
				:token="authToken"
				@select-user="onUserSelected"
			/>

			<!-- Create group panel -->
			<CreateGroup
				v-if="showCreateGroup"
				:token="authToken"
				:current-username="currentUsername"
				@created="onGroupCreated"
			/>

			<!-- Settings panel -->
			<UserSettings
				v-if="showSettings"
				:token="authToken"
				@logout="onUserLogout"
				@username-changed="onUsernameChanged"
			/>

			<!-- Error -->
			<ErrorMsg v-if="errormsg" :msg="errormsg" />

			<!-- Conversations list -->
			<div class="flex-grow-1 overflow-auto">
				<div v-if="loading" class="text-center mt-3"><LoadingSpinner /></div>
				<div
					v-for="conv in conversations"
					:key="conv.id"
					class="d-flex align-items-center px-3 py-2 border-bottom conversation-item"
					:class="{ 'bg-white': selectedConversationId === conv.id }"
					style="cursor: pointer;"
					@click="selectConversation(conv.id)"
				>
					<div class="me-2 flex-shrink-0">
						<img v-if="conv.photoUrl" :src="photoSrc(conv.photoUrl)" class="rounded-circle" style="width:46px; height:46px; object-fit:cover;" />
						<div v-else class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white fw-bold" style="width:46px; height:46px; font-size:18px;">
							{{ conv.name ? conv.name[0].toUpperCase() : '?' }}
						</div>
					</div>
					<div class="flex-grow-1 overflow-hidden">
						<div class="d-flex justify-content-between align-items-center">
							<span class="fw-semibold text-truncate">{{ conv.name }}</span>
							<span class="text-muted small ms-2 flex-shrink-0">{{ conv.lastMessage ? formatTime(conv.lastMessage.timestamp) : '' }}</span>
						</div>
						<div class="d-flex justify-content-between align-items-center">
							<span class="text-muted small text-truncate">
								<span v-if="conv.lastMessage && conv.lastMessage.isPhoto">📷 Photo</span>
								<span v-else-if="conv.lastMessage">{{ conv.lastMessage.preview }}</span>
								<span v-else class="fst-italic">No messages yet</span>
							</span>
							<span v-if="conv.unreadCount > 0" class="badge rounded-pill bg-primary ms-2 flex-shrink-0" style="font-size:11px;">
								{{ conv.unreadCount }}
							</span>
						</div>
					</div>
				</div>
				<div v-if="conversations.length === 0 && !loading" class="text-center text-muted mt-5 px-3">
					<p>No conversations yet.</p>
					<p class="small">Search for a user to start chatting.</p>
				</div>
			</div>
		</div>

		<!-- ── RIGHT PANEL ────────────────────────────────────────────────── -->
		<div class="flex-grow-1 d-flex flex-column bg-light">

			<!-- No conversation selected -->
			<div v-if="!selectedConversationId && !pendingUsername" class="flex-grow-1 d-flex align-items-center justify-content-center text-muted">
				<p class="fs-5">Select a conversation to start chatting</p>
			</div>

			<!-- Pending new conversation (user selected from search) -->
			<template v-else-if="pendingUsername">
				<div class="d-flex align-items-center px-3 py-2 border-bottom bg-white">
					<div class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white fw-bold me-2" style="width:38px; height:38px; font-size:16px;">
						{{ pendingUsername[0].toUpperCase() }}
					</div>
					<span class="fw-semibold">{{ pendingUsername }}</span>
				</div>
				<div class="flex-grow-1 d-flex align-items-center justify-content-center text-muted">
					<p class="small">Send your first message to start the conversation.</p>
				</div>
				<!-- Message input -->
				<div class="border-top bg-white px-3 py-2">
					<div class="d-flex gap-2">
						<input
							v-model="messageText"
							type="text"
							class="form-control"
							placeholder="Type a message..."
							:disabled="sendingMessage"
							@keyup.enter="sendMessage"
						/>
						<label class="btn btn-outline-secondary mb-0" title="Send image">
							📷
							<input type="file" accept="image/*" class="d-none" @change="onImageSelected" />
						</label>
						<button class="btn btn-primary" :disabled="sendingMessage || (!messageText.trim() && !messageImageFile)" @click="sendMessage">
							<LoadingSpinner v-if="sendingMessage" />
							<span v-else>Send</span>
						</button>
					</div>
				</div>
			</template>

			<!-- Existing conversation -->
			<template v-else>
				<!-- Chat header -->
				<div class="d-flex align-items-center px-3 py-2 border-bottom bg-white">
					<div class="me-2">
						<img v-if="selectedConversation && selectedConversation.photoUrl" :src="photoSrc(selectedConversation.photoUrl)" class="rounded-circle" style="width:38px; height:38px; object-fit:cover;" />
						<div v-else class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white fw-bold" style="width:38px; height:38px; font-size:16px;">
							{{ selectedConversation ? selectedConversation.name[0].toUpperCase() : '?' }}
						</div>
					</div>
					<span class="fw-semibold">{{ selectedConversation ? selectedConversation.name : '' }}</span>
					<LoadingSpinner v-if="conversationLoading" class="ms-2" />
					<button v-if="isGroupChat" class="btn btn-sm btn-link ms-auto p-0 text-muted" style="font-size:18px;" title="Group settings" @click="showGroupSettings = !showGroupSettings">⚙️</button>
				</div>

				<!-- Group settings panel — rendered by GroupSettings component -->
				<GroupSettings
					v-if="isGroupChat && showGroupSettings"
					:group-id="selectedConversationId"
					:participants="currentConversation ? currentConversation.participants : []"
					:token="authToken"
					@updated="onGroupUpdated"
					@left="onGroupLeft"
				/>

				<!-- Messages area -->
				<div class="flex-grow-1 overflow-auto px-3 py-2" style="display: flex; flex-direction: column; gap: 6px;">
					<ErrorMsg v-if="conversationError" :msg="conversationError" />

					<div
						v-for="msg in sortedMessages"
						:key="msg.id"
						class="d-flex"
						:class="msg.senderUsername === currentUsername ? 'justify-content-end' : 'justify-content-start'"
					>
						<div
							class="px-3 py-2 rounded-3"
							style="max-width: 65%; word-break: break-word; position: relative;"
							:style="msg.senderUsername === currentUsername ? 'background:#dcf8c6;' : 'background:#ffffff;'"
						>
							<!-- Forwarded marker -->
							<div v-if="msg.isForwarded" class="small text-muted mb-1 d-flex align-items-center gap-1">
								<span>↪</span><span class="fst-italic">Forwarded</span>
							</div>

							<!-- Sender name (groups only) -->
							<div v-if="currentConversation && currentConversation.type === 'group' && msg.senderUsername !== currentUsername" class="small fw-bold text-primary mb-1">
								{{ msg.senderUsername }}
							</div>

							<!-- Reply preview -->
							<div v-if="msg.replyTo" class="border-start border-primary ps-2 mb-1 small text-muted">
							  <span class="fw-semibold">{{ msg.replyTo.senderUsername }}</span><br />
							  
							  <!-- Show resized photo if content is a photo -->
							  <img v-if="msg.replyTo.content && msg.replyTo.content.startsWith('/photos/')" 
							     :src="photoSrc(msg.replyTo.content)" 
							     class="rounded mt-1" 
							     style="max-width: 60px; max-height: 60px; object-fit: cover; display: block;" />
							     
							  <!-- Otherwise show resized text -->
							  <span v-else>{{ msg.replyTo.content }}</span>
							</div>

							<!-- Content -->
							<img v-if="msg.photoUrl" :src="photoSrc(msg.photoUrl)" class="rounded" style="max-width:200px; max-height:200px; display:block;" />
							<span v-else>{{ msg.content }}</span>

							<!-- Footer: time + checkmarks + reply button -->
							<div class="d-flex justify-content-end align-items-center gap-2 mt-1">
								<span class="text-muted" style="font-size:11px;">{{ formatTime(msg.timestamp) }}</span>
								<span v-if="msg.senderUsername === currentUsername" class="text-muted" style="font-size:11px;">{{ checkmarkIcon(msg.status) }}</span>
								<button class="btn btn-link btn-sm p-0 text-muted" style="font-size:11px;" title="Reply" @click="setReply(msg)">↩</button>
								<button class="btn btn-link btn-sm p-0 text-muted" style="font-size:11px;" title="Forward" @click="toggleForwardPicker(msg.id)">↪</button>
								<button
									v-if="msg.senderUsername === currentUsername"
									class="btn btn-link btn-sm p-0 text-danger"
								    style="font-size:11px;"
								    title="Delete"
								    @click="deleteMessage(selectedConversationId, msg.id)"
								>🗑</button>
							</div>

							<!-- Reactions bar -->
							<div class="d-flex flex-wrap align-items-center gap-1 mt-1">
								<!-- Existing reactions grouped by emoticon -->
								<span
									v-for="r in msg.reactions"
									:key="r.username"
									class="badge rounded-pill"
									:class="r.username === currentUsername ? 'bg-primary' : 'bg-secondary'"
									style="font-size:13px; cursor:pointer;"
									:title="r.username"
									@click="r.username === currentUsername ? removeReaction(msg.id) : null"
								>{{ r.emoticon }}</span>

								<!-- Add reaction button -->
								<button
									class="btn btn-link btn-sm p-0 text-muted"
									style="font-size:13px;"
									title="Add reaction"
									@click="toggleEmojiPicker(msg.id)"
								>😊</button>

								<!-- Emoji picker -->
								<div v-if="showEmojiPicker === msg.id" class="bg-white border rounded p-1 d-flex gap-1 flex-wrap" style="position:absolute; z-index:100; margin-top:4px;">
									<span
										v-for="emoji in ['👍','❤️','😂','😮','😢','🔥','👏','🎉']"
										:key="emoji"
										style="cursor:pointer; font-size:18px;"
										@click="addReaction(msg.id, emoji)"
									>{{ emoji }}</span>
								</div>

								<!-- Forward picker -->
								<div v-if="showForwardPicker === msg.id" class="bg-white border rounded p-2" style="position:absolute; z-index:100; min-width:200px; max-height:200px; overflow-y:auto;">
									<div class="small fw-bold mb-1">Forward to...</div>
									<div
										v-for="conv in conversations"
										:key="conv.id"
										class="d-flex align-items-center gap-2 py-1 px-1 rounded"
										style="cursor:pointer;"
										:style="conv.id === selectedConversationId ? 'background:#f0f0f0;' : ''"
										@click="forwardMessage(msg.id, conv.id)"
									>
										<div class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white flex-shrink-0" style="width:24px; height:24px; font-size:11px;">
											{{ conv.name ? conv.name[0].toUpperCase() : '?' }}
										</div>
										<span class="small text-truncate">{{ conv.name }}</span>
									</div>
									<div v-if="forwardError" class="text-danger small mt-1">{{ forwardError }}</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Reply preview bar -->
				<div v-if="replyToMessage" class="px-3 py-1 border-top bg-white d-flex align-items-center justify-content-between">
					<div class="small text-muted">
						<span class="fw-semibold">Replying to {{ replyToMessage.senderUsername }}:</span>
						{{ replyToMessage.content || '📷 Photo' }}
					</div>
					<button class="btn btn-sm btn-link text-danger p-0" @click="cancelReply">✕</button>
				</div>

				<!-- Message input -->
				<div class="border-top bg-white px-3 py-2">
					<div class="d-flex gap-2">
						<input
							v-model="messageText"
							type="text"
							class="form-control"
							placeholder="Type a message..."
							:disabled="sendingMessage"
							@keyup.enter="sendMessage"
						/>
						<label class="btn btn-outline-secondary mb-0" title="Send image">
							📷
							<input type="file" accept="image/*" class="d-none" @change="onImageSelected" />
						</label>
						<button class="btn btn-primary" :disabled="sendingMessage || (!messageText.trim() && !messageImageFile)" @click="sendMessage">
							<LoadingSpinner v-if="sendingMessage" />
							<span v-else>Send</span>
						</button>
					</div>
					<div v-if="messageImageFile" class="small text-muted mt-1">📷 {{ messageImageFile.name }} <button class="btn btn-link btn-sm p-0 text-danger" @click="messageImageFile = null">✕</button></div>
				</div>
			</template>
		</div>

	</div>
</template>

<style scoped>
.conversation-item:hover {
	background-color: #ebebeb !important;
}
</style>
