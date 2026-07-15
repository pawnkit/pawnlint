native bad_api();

new const DEBUG = 1;

enum Status
{
	UNKNOWN_STATE
}

stock ProcessData(Float:value, foo)
{
	new temp_buffer;
	new bar;
	return temp_buffer + bar + foo + _:value;
}

main()
{
	ProcessData(Float:1, 0);
}
