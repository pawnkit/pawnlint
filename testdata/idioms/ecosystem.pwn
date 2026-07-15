hook OnPlayerConnect(playerid)
{
	return 1;
}

hook OnPlayerDisconnect(playerid, reason)
{
	stop gReconnectTimer[playerid];
	return 1;
}

timer HeartbeatTimer[0](playerid)
{
	return 1;
}

stock LoadConfig()
{
	new File:file;
	file = fopen("config.ini", io_read);
	if (file)
	{
		new line[128];
		fread(file, line);
		fclose(file);
	}
	return 1;
}

stock LoadConfigOrBail(filename[])
{
	new File:file;
	file = fopen(filename, io_read);
	if (!file)
	{
		return 0;
	}
	fclose(file);
	return 1;
}

stock FireWeapon(playerid, weaponid, hittype = -1, hitid = -1, Float:fX = 0.0)
{
	#pragma unused hittype, hitid, fX
	return playerid + weaponid;
}

ShowSomeDialog(playerid)
{
	new ret;

	inline Response(dialogid, response, listitem, string:inputtext[])
	{
		#pragma unused dialogid, listitem, inputtext

		if (response)
		{
			ret = ProcessChoice(playerid);

			if (ret == 0)
			{
				return 0;
			}
		}

		return 1;
	}

	return DialogShowCallback(playerid, 0, "Title", "Body", "Ok", "Cancel");
}
