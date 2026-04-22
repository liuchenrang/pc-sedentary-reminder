export namespace main {
	
	export class ShortcutConfig {
	    resetKey: string;
	    closeNotifyKey: string;
	    ctrl: boolean;
	    shift: boolean;
	    alt: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShortcutConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resetKey = source["resetKey"];
	        this.closeNotifyKey = source["closeNotifyKey"];
	        this.ctrl = source["ctrl"];
	        this.shift = source["shift"];
	        this.alt = source["alt"];
	    }
	}
	export class SitLongConfig {
	    interval: number;
	    isRunning: boolean;
	    remaining: number;
	    shortcut: ShortcutConfig;
	    notificationDuration: number;
	    activateOnTimer: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SitLongConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.interval = source["interval"];
	        this.isRunning = source["isRunning"];
	        this.remaining = source["remaining"];
	        this.shortcut = this.convertValues(source["shortcut"], ShortcutConfig);
	        this.notificationDuration = source["notificationDuration"];
	        this.activateOnTimer = source["activateOnTimer"];
	        this.message = source["message"];
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

}

