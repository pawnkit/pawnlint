native bad_native();

new const max_clients = 32;
new globalScore;
new Timer:g_round_Timer;

enum player_state
{
	player_none
}

stock load_player(Float:speed, BadParameter)
{
	new BadLocal;
BadLabel:
	return BadLocal + BadParameter + _:speed;
}

main()
{
	load_player(Float:1, 0);
}
