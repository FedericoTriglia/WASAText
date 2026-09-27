<script>
export default {
	name: 'UserSettings',
	props: {
		token: { type: String, required: true },
	},
	emits: ['logout'],
	data() {
		return {
			newPhotoFile: null,
			error: null,
			success: null,
		}
	},
	computed: {
		currentUsername() {
			return localStorage.getItem('username') || '';
		},
	},
	methods: {
		authHeader() {
			return { Authorization: 'Bearer ' + this.token };
		},
		onPhotoSelected(event) {
			this.newPhotoFile = event.target.files[0] || null;
		},
		async updatePhoto() {
			if (!this.newPhotoFile) return;
			this.error = null;
			this.success = null;
			const formData = new FormData();
			formData.append('photo', this.newPhotoFile);
			try {
				await this.$axios.put('/users/me/photo', formData, {
					headers: { ...this.authHeader(), 'Content-Type': 'multipart/form-data' },
				});
				this.success = 'Photo updated successfully.';
				this.newPhotoFile = null;
			} catch (e) {
				this.error = e.toString();
			}
		},
		logout() {
			localStorage.removeItem('token');
			localStorage.removeItem('username');
			this.$emit('logout');
		},
	},
}
</script>

<template>
	<div class="px-3 py-2 border-bottom bg-white">
		<div class="small fw-bold mb-2">Settings — {{ currentUsername }}</div>
		<div class="mb-2">
			<input type="file" accept="image/*" class="form-control form-control-sm" @change="onPhotoSelected" />
			<button class="btn btn-sm btn-primary mt-1 w-100" @click="updatePhoto">Change photo</button>
		</div>
		<div v-if="error" class="text-danger small">{{ error }}</div>
		<div v-if="success" class="text-success small">{{ success }}</div>
		<button class="btn btn-sm btn-outline-danger w-100 mt-2" @click="logout">Logout</button>
	</div>
</template>
