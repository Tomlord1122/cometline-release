<script lang="ts">
</script>

<div class="empty-state">
	<div class="avatar rounded-full border border-gray-400" aria-hidden="true">
		<img
			src="/project_avatar_192.png"
			srcset="/project_avatar_96.png 96w, /project_avatar_192.png 192w, /project_avatar_384.png 384w"
			sizes="82px"
			alt=""
		/>
	</div>
	<p class="subtitle">A thought, a task, a file — Cometline continues.</p>
</div>

<style>
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
	}

	.avatar {
		width: 82px;
		height: 82px;
		background: #fff;
		margin-bottom: 24px;
		box-shadow: var(--shadow-card);
		overflow: hidden;
	}

	.avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: 50%;
		display: block;
	}

	.subtitle {
		font-size: 14px;
		color: var(--text-muted);
		margin: 0;
	}
</style>
