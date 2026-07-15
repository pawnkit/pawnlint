enum PlayerState
{
    STATE_NONE,
    STATE_ALIVE,
    STATE_DEAD
}

CheckState(PlayerState:current)
{
	switch (current)
	{
		case STATE_NONE:
			return 0;
	}
	return 1;
}
