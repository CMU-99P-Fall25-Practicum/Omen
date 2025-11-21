export namespace main {
	
	export enum PropModel {
	    Friis = "Friis",
	    LogDistance = "LogDistance",
	    LogNormalShadowing = "LogNormalShadowing",
	}
	export enum WifiMode {
	    a = "a",
	    b = "b",
	    g = "g",
	    n = "n",
	    ax = "ax",
	    ac = "ac",
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
	export class PropagationModel {
	    model: string;
	    exp: number;
	    s: number;
	
	    static createFrom(source: any = {}) {
	        return new PropagationModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.exp = source["exp"];
	        this.s = source["s"];
	    }
	}
	export class Nets {
	    noise_th: number;
	    propagation_model: PropagationModel;
	
	    static createFrom(source: any = {}) {
	        return new Nets(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.noise_th = source["noise_th"];
	        this.propagation_model = this.convertValues(source["propagation_model"], PropagationModel);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	export class Test {
	    name: string;
	    type: string;
	    timeframe: number;
	    node: string;
	    position: string;
	
	    static createFrom(source: any = {}) {
	        return new Test(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.timeframe = source["timeframe"];
	        this.node = source["node"];
	        this.position = source["position"];
	    }
	}

}

