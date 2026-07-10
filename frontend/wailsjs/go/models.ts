export namespace app {
	
	export class VMData {
	    id: string;
	    name: string;
	    provider: string;
	    region: string;
	    state: string;
	    platform: string;
	    size: string;
	    privateIP: string;
	    publicIP: string;
	    osUser: string;
	    tags: string[];
	    canConnect: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VMData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.region = source["region"];
	        this.state = source["state"];
	        this.platform = source["platform"];
	        this.size = source["size"];
	        this.privateIP = source["privateIP"];
	        this.publicIP = source["publicIP"];
	        this.osUser = source["osUser"];
	        this.tags = source["tags"];
	        this.canConnect = source["canConnect"];
	    }
	}

}
