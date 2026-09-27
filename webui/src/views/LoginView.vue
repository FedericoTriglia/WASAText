<script>
export default {
	data() {
		return {
			username: '',
			errormsg: null,
			loading: false,
		}
	},
	methods: {
		async doLogin() {
			// Validate username format before sending the request
			const pattern = /^[A-Za-z0-9]{3,16}$/;
			if (!pattern.test(this.username)) {
				this.errormsg = 'Username must be 3-16 alphanumeric characters.';
				return;
			}

			this.loading = true;
			this.errormsg = null;

			try {
				const response = await this.$axios.post('/session', {
					name: this.username,
				});

				// Save the token in localStorage for subsequent authenticated requests
				localStorage.setItem('token', response.data.identifier);
				localStorage.setItem('username', this.username);

				// Redirect to the conversations list
				this.$router.push('/conversations');
			} catch (e) {
				this.errormsg = e.toString();
			}

			this.loading = false;
		},
	},
}
</script>

<template>
	<div class="d-flex justify-content-center align-items-center" style="min-height: 100vh;">
		<div class="card shadow" style="width: 100%; max-width: 400px;">
			<div class="card-body p-4">
				<h2 class="card-title text-center mb-4">WASAText</h2>
				<p class="text-muted text-center mb-4">Enter your username to log in or create an account.</p>

				<ErrorMsg v-if="errormsg" :msg="errormsg" />

				<div class="mb-3">
					<label for="username" class="form-label">Username</label>
					<input
						id="username"
						v-model="username"
						type="text"
						class="form-control"
						placeholder="e.g. Maria"
						:disabled="loading"
						@keyup.enter="doLogin"
					/>
					<div class="form-text">3-16 alphanumeric characters.</div>
				</div>

				<button
					class="btn btn-primary w-100"
					:disabled="loading || username.length === 0"
					@click="doLogin"
				>
					<LoadingSpinner v-if="loading" />
					<span v-else>Login</span>
				</button>
			</div>
		</div>
	</div>
</template>

<style scoped>
</style>
