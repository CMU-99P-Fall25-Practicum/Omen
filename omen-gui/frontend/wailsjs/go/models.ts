export namespace main {
	
	export enum PropModel {
	    Friis = "Friis",
	    LogDistance = "LogDistance",
	    LogNormalShadowing = "LogNormalShadowing",
	}
	export class AP {
	    id: string;
	    mode: string;
	    channel: number;
	    ssid: string;
	    position: string;
	
	    static createFrom(source: any = {}) {
	        return new AP(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.mode = source["mode"];
	        this.channel = source["channel"];
	        this.ssid = source["ssid"];
	        this.position = source["position"];
	    }
	}
	export class Sta {
	    id: string;
	    position: string;
	
	    static createFrom(source: any = {}) {
	        return new Sta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.position = source["position"];
	    }
	}

}

