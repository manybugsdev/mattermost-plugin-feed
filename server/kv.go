package main

const KVKey = "dev.manybugs.feed"

func (p *Plugin) SaveFeeds(feeds []Feed) (bool, error) {
	return p.client.KV.Set(KVKey, feeds)
}

func (p *Plugin) LoadFeeds() []Feed {
	feeds := []Feed{}
	err := p.client.KV.Get(KVKey, &feeds)
	if err != nil {
		p.client.Log.Error("Error loading feeds: " + err.Error())
	}
	return feeds
}
