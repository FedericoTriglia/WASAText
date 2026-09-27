<script>
export default {
	name: 'CreateGroup',
	props: {
		token: { type: String, required: true },
		currentUsername: { type: String, required: true },
	},
	emits: ['created'],
	data() {
		return {
			newGroupName: '',
			newGroupMemberQuery: '',
			newGroupMemberResults: [],
			newGroupMemberSearching: false,
			newGroupMembers: [],
			error: null,
			creating: false,
		}
	},
	methods: {
		authHeader() {
			return { Authorization: 'Bearer ' + this.token };
		},
		async searchGroupMembers() {
			if (this.newGroupMemberQuery.length < 3) {
				this.newGroupMemberResults = [];
				return;
			}
			this.newGroupMemberSearching = true;
			try {
				const response = await this.$axios.get('/users', {
					headers: this.authHeader(),
					params: { randomUser: this.newGroupMemberQuery },
				});
				this.newGroupMemberResults = (response.data || []).filter(u =>
					u.username !== this.currentUsername &&
					!this.newGroupMembers.includes(u.username)
				);
			} catch (e) {
				// Ignore
			}
			this.newGroupMemberSearching = false;
		},
		addGroupMember(username) {
			if (!this.newGroupMembers.includes(username)) {
				this.newGroupMembers.push(username);
			}
			this.newGroupMemberQuery = '';
			this.newGroupMemberResults = [];
		},
		removeGroupMember(username) {
			this.newGroupMembers = this.newGroupMembers.filter(m => m !== username);
		},
		async createGroup() {
			this.error = null;
			if (!this.newGroupName.trim()) {
				this.error = 'Group name is required.';
				return;
			}
			if (this.newGroupMembers.length < 1) {
				this.error = 'Add at least one member.';
				return;
			}
			this.creating = true;
			try {
				const response = await this.$axios.post('/groups', {
					name: this.newGroupName.trim(),
					memberUsernames: this.newGroupMembers,
				}, { headers: this.authHeader() });
				this.newGroupName = '';
				this.newGroupMembers = [];
				this.$emit('created', response.data.groupId);
			} catch (e) {
				this.error = e.response && e.response.status === 404
					? 'One or more members do not exist.'
					: e.toString();
			}
			this.creating = false;
		},
	},
}
</script>

<template>
	<div class="px-3 py-2 border-bottom bg-white">
		<div class="small fw-bold mb-2">New Group</div>
		<input v-model="newGroupName" type="text" class="form-control form-control-sm mb-2" placeholder="Group name (3-50 chars)" />
		<input v-model="newGroupMemberQuery" type="text" class="form-control form-control-sm" placeholder="Search members (min 3 chars)..." @input="searchGroupMembers" />
		<div v-if="newGroupMemberSearching" class="text-center mt-1"><LoadingSpinner /></div>
		<ul v-if="newGroupMemberResults.length > 0" class="list-group mt-1 mb-2">
			<li v-for="user in newGroupMemberResults" :key="user.username" class="list-group-item list-group-item-action py-1 small" style="cursor:pointer;" @click="addGroupMember(user.username)">+ {{ user.username }}</li>
		</ul>
		<div v-if="newGroupMembers.length > 0" class="d-flex flex-wrap gap-1 mb-2">
			<span v-for="m in newGroupMembers" :key="m" class="badge bg-primary d-flex align-items-center gap-1">{{ m }} <button class="btn-close btn-close-white" style="font-size:8px;" @click="removeGroupMember(m)"></button></span>
		</div>
		<div v-if="error" class="text-danger small mb-1">{{ error }}</div>
		<button class="btn btn-sm btn-primary w-100" :disabled="creating" @click="createGroup">
			<LoadingSpinner v-if="creating" /><span v-else>Create Group</span>
		</button>
	</div>
</template>
