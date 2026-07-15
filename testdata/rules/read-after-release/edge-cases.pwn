native Resource:Acquire();
native Release(Resource:resource);
native Consume(Resource:resource);

main()
{
}

stock EscapeBeforeRelease()
{
	new Resource:resource = Acquire();
	Custom(resource);
	Release(resource);
	Consume(resource);
}

stock BranchReassignment(bool:replace)
{
	new Resource:resource = Acquire();
	Release(resource);
	if (replace)
	{
		resource = Acquire();
		Consume(resource);
		Release(resource);
	}
}

stock Custom(Resource:resource)
{
	return _:resource;
}
