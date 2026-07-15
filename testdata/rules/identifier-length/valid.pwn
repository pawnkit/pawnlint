new globalScore;

enum PlayerState
{
	PLAYER_STATE_NONE,
	PLAYER_STATE_ACTIVE
}

stock LoadPlayer(playerId)
{
	new score = playerId;
	for (new i = 0; i < 2; i++)
	{
		score += i;
	}
	return score;
}

main()
{
	LoadPlayer(1);
}
