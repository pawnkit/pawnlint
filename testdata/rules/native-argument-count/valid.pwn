main()
{
    new version[24];
    Kick(0);
    GetPlayerVersion(0, version);
    SendRconCommand("echo ready");
    print("Directory '"DIRECTORY_ROOT DIRECTORY_MAIN"' is ready.");
}
