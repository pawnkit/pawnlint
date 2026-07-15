new bool:isEnabled;

bool:IsReady()
{
	return true;
}

stock UpdatePlayer(bool:hasAccess)
{
	new bool:canContinue = hasAccess;
	new bool:b_visible = isEnabled;
	new island = 1;
	return canContinue && b_visible && island;
}

main()
{
	UpdatePlayer(IsReady());
}
