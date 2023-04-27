# chilctl
Command-line tool to adjust settings on [Chiltrix CX34](https://www.chiltrix.com/) heatpumps
over RS-485 / Modbus. This tool is built from the exceptional work at https://github.com/gonzojive/heatpump
which offers a full-featured data collector and cloud integration. This tool focuses
on the CLI experience to interact with the heatpump.

# Getting Started

A heatpump. Chiltrix makes an fantastic product, I will talk your ear off about it.

An RS-485 adapter. I prefer USB devices with FT232L chipset because the drivers work out of the box
on Linux, whereas the CH343G chipset driver is out of tree. The on-board UART on a Raspberry Pi should
work with an appropriate signaling adapter.

Twisted pair wire. RS-485 is super resilient and could probably run short distances over a pair
of coat hangers, but your best results will be 22 AWG twisted pair. Old "Cat 3" phone wire is perfect,
we're only running at 9600 baud here.

Go compiler. Like this:

    go install github.com/sodabrew/chilctl

# FAQ

## Do I need to dig up my lawn to install a heat pump?

No! Well, yes if you install a geothermal heat pump. Modern air heat pumps can operate efficiently
even at below-freezing air temperatures, so you can install the outdoor unit in almost any climate.

## Do all heatpumps have ugly high-wall units?

No! I installed my system using ducted air handlers, they dropped right in where the old gas furnace was.

## Do all heatpumps require "freon" refrigerant lines in every room?

No! Chiltrix is an air-to-water heatpump. Like a traditional boiler, water pipes carry hot (or cold!)
water to each station in your home or building. Station can use an air handler, wall unit, radiant pipes, or mix and match.
Note that you MUST fully and carefully insulate your pipes in cooling mode to prevent condensation.

## What's a heatpump?
Heat pumps use a compression cycle to move heat around. In a home refrigerator, heat is moved
from inside the food compartment out to the surrounding room. This keeps your food cold.
A heat pump can run in both directions: taking heat from outside and moving it into your
home in the winter, or taking heat from inside your home and moving it outside in the summer. 

Traditional electric heaters are nearly 100% efficient, meaning that all of the energy input
turns into heat output. Heat pumps are typically "400% efficient" meaning that for every
1 unit of electrical energy input, 4 units of heat energy are transported. This is
called the [Coefficient of Performance](https://en.wikipedia.org/wiki/Coefficient_of_performance).
Modern heat pumps since in the 2000's are even able to operate at below-freezing temperatures --
that is, they can actually pull heat from zero degree air and move it into your home!

## What are RS-485, Modbus?

[RS-485](https://en.wikipedia.org/wiki/RS-485) is a two-wire differential serial line that
can connect multiple devices to the same line. [Modbus](https://en.wikipedia.org/wiki/Modbus) is
a standard for communicating with industrial equipment and is often used in residential HVAC
and swimming pool equipment.

Good luck!
