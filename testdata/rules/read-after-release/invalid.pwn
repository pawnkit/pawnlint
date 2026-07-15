native Resource:Acquire();
native Release(Resource:resource);
native Consume(Resource:resource);

main()
{
	new Resource:resource = Acquire();
	Release(resource);
	Consume(resource);
	if (resource)
	{
		Release(resource);
	}
}

stock Conditional(bool:release)
{
	new Resource:resource = Acquire();
	if (release)
	{
		Release(resource);
	}
	Consume(resource);
}

stock Loop()
{
	new Resource:resource = Acquire();
	while (resource)
	{
		Release(resource);
	}
}
