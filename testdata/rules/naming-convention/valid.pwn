native N_Load();

new const MAX_CLIENTS = 32;
new g_score;
new Timer:g_roundTimer;

enum PlayerState
{
	PLAYER_STATE_NONE,
	PLAYER_STATE_ACTIVE
}

public bad_callback()
{
	return 1;
}

stock LoadPlayer(Float:f_speed, ignored_parameter)
{
	new playerCount = _:f_speed;
good_label:
	return playerCount + ignored_parameter;
}

main()
{
	LoadPlayer(Float:1, 0);
}
