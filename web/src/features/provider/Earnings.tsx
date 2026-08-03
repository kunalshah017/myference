import { Money } from '../../components/Money'

export function Earnings({ earned, claimable }: { earned:string; claimable:string }) { return <div className="earnings-proof"><strong><Money wei={earned}/> earned</strong><span><Money wei={claimable}/> currently claimable</span></div> }
