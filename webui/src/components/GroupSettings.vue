<script>
export default {
	name: 'GroupSettings',
	props: {
		groupId: { type: Number, required: true },
		participants: { type: Array, default: () => [] },
		token: { type: String, required: true },
	},
	emits: ['updated', 'left'],
	data() {
		return {
			editGroupName: '',
			editGroupPhotoFile: null,
			error: null,
			success: null,
			addMemberQuery: '',
			addMemberResults: [],
			addMemberSearching: false,
			addMemberError: null,
			addMemberSuccess: null,
			leavingGroup: false,
		}
	},
	methods: {
		authHeader() {
			return { Authorization: 'Bearer ' + this.token };
		},
		async updateGroupName() {
			this.error = null;
			this.success = null;
			if (!this.editGroupName.trim()) {
				this.error = 'Name cannot be empty.';
				return;
			}
			try {
				await this.$axios.put('/groups/' + this.groupId + '/name', {
					name: this.editGroupName.trim(),
				}, { headers: this.authHeader() });
				this.success = 'Group name updated.';
				this.editGroupName = '';
				this.$emit('updated');
			} catch (e) {
				this.error = e.toString();
			}
		},
		onGroupPhotoSelected(event) {
			this.editGroupPhotoFile = event.target.files[0] || null;
		},
		async updateGroupPhoto() {
			if (!this.editGroupPhotoFile) return;
			this.error = null;
			this.success = null;
			const formData = new FormData();
			formData.append('photo', this.editGroupPhotoFile);
			try {
				await this.$axios.put('/groups/' + this.groupId + '/photo', formData, {
					headers: { ...this.authHeader(), 'Content-Type': 'multipart/form-data' },
				});
				this.success = 'Group photo updated.';
				this.editGroupPhotoFile = null;
				this.$emit('updated');
			} catch (e) {
				this.error = e.toString();
			}
		},
		async searchAddMember() {
			if (this.addMemberQuery.length < 3) {
				this.addMemberResults = [];
				return;
			}
			this.addMemberSearching = true;
			try {
				const response = await this.$axios.get('/users', {
					headers: this.authHeader(),
					params: { randomUser: this.addMemberQuery },
				});
				this.addMemberResults = (response.data || []).filter(u => !this.participants.includes(u.username));
			} catch (e) {
				// Ignore
			}
			this.addMemberSearching = false;
		},
		async addToGroup(username) {
			this.addMemberError = null;
			this.addMemberSuccess = null;
			try {
				await this.$axios.post('/groups/' + this.groupId + '/members', {
					username: username,
				}, { headers: this.authHeader() });
				this.addMemberSuccess = username + ' added to the group.';
				this.addMemberQuery = '';
				this.addMemberResults = [];
				this.$emit('updated');
			} catch (e) {
				if (e.response && e.response.status === 409) {
					this.addMemberError = username + ' is already a member.';
				} else if (e.response && e.response.status === 404) {
					this.addMemberError = 'User not found.';
				} else {
					this.addMemberError = e.toString();
				}
			}
		},
		async leaveGroup() {
			if (!confirm('Are you sure you want to leave this group?')) return;
			this.leavingGroup = true;
			try {
				await this.$axios.delete('/groups/' + this.groupId + '/members/me', {
					headers: this.authHeader(),
				});
				this.$emit('left');
			} catch (e) {
				this.error = e.toString();
			}
			this.leavingGroup = false;
		},
	},
}
</script>

<template>
	<div class="px-3 py-2 border-bottom bg-white">
		<div class="small fw-bold mb-2">Group Settings</div>
		<div class="mb-2">
			<label class="form-label small">Change group name</label>
			<input v-model="editGroupName" type="text" class="form-control form-control-sm" placeholder="New name (3-50 chars)" />
			<button class="btn btn-sm btn-primary mt-1 w-100" @click="updateGroupName">Change name</button>
		</div>
		<div class="mb-2">
			<label class="form-label small">Change group photo</label>
			<input type="file" accept="image/*" class="form-control form-control-sm" @change="onGroupPhotoSelected" />
			<button class="btn btn-sm btn-primary mt-1 w-100" @click="updateGroupPhoto">Change photo</button>
		</div>
		<div class="mb-2">
			<label class="form-label small">Add member</label>
			<input v-model="addMemberQuery" type="text" class="form-control form-control-sm" placeholder="Search user (min 3 chars)..." @input="searchAddMember" />
			<div v-if="addMemberSearching" class="text-center mt-1"><LoadingSpinner /></div>
			<ul v-if="addMemberResults.length > 0" class="list-group mt-1">
				<li v-for="user in addMemberResults" :key="user.username" class="list-group-item list-group-item-action py-1 small" style="cursor:pointer;" @click="addToGroup(user.username)">+ {{ user.username }}</li>
			</ul>
			<div v-if="addMemberError" class="text-danger small mt-1">{{ addMemberError }}</div>
			<div v-if="addMemberSuccess" class="text-success small mt-1">{{ addMemberSuccess }}</div>
		</div>
		<div v-if="error" class="text-danger small">{{ error }}</div>
		<div v-if="success" class="text-success small">{{ success }}</div>
		<button class="btn btn-sm btn-outline-danger w-100 mt-2" :disabled="leavingGroup" @click="leaveGroup">
			<LoadingSpinner v-if="leavingGroup" /><span v-else>Leave group</span>
		</button>
	</div>
</template>
