<script>
export default {
	name: 'SearchUsers',
	props: {
		token: { type: String, required: true },
	},
	emits: ['select-user'],
	data() {
		return {
			searchQuery: '',
			searchResults: [],
			searching: false,
			error: null,
		}
	},
	methods: {
		authHeader() {
			return { Authorization: 'Bearer ' + this.token };
		},
		async searchUsers() {
			if (this.searchQuery.length < 3) {
				this.searchResults = [];
				return;
			}
			this.searching = true;
			this.error = null;
			try {
				const response = await this.$axios.get('/users', {
					headers: this.authHeader(),
					params: { randomUser: this.searchQuery },
				});
				this.searchResults = response.data || [];
			} catch (e) {
				this.error = e.toString();
			}
			this.searching = false;
		},
		selectUser(username) {
			this.searchQuery = '';
			this.searchResults = [];
			this.$emit('select-user', username);
		},
	},
}
</script>

<template>
	<div class="px-3 py-2 border-bottom bg-white">
		<input v-model="searchQuery" type="text" class="form-control form-control-sm" placeholder="Search users (min 3 chars)..." @input="searchUsers" />
		<div v-if="searching" class="text-center mt-1"><LoadingSpinner /></div>
		<div v-if="error" class="text-danger small mt-1">{{ error }}</div>
		<ul v-if="searchResults.length > 0" class="list-group mt-1">
			<li v-for="user in searchResults" :key="user.username" class="list-group-item list-group-item-action d-flex align-items-center gap-2 py-1" style="cursor:pointer;" @click="selectUser(user.username)">
				<img v-if="user.photoUrl" :src="user.photoUrl" class="rounded-circle" style="width:28px; height:28px; object-fit:cover;" />
				<div v-else class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white" style="width:28px; height:28px; font-size:12px;">
					{{ user.username[0].toUpperCase() }}
				</div>
				<span class="small">{{ user.username }}</span>
			</li>
		</ul>
	</div>
</template>
