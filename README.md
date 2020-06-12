# Spring Model

A web of point-masses connected by springs is modeled as a graph and physics calculation.
Each springs ends accelerates its mass to reach the springs ideal length.
The force effected in this way on a springs masses is given by a factor K specific to the spring and the distance between the current positions of the two connected masses.
The web is represented by a non-cyclic graph of connected point-masses.
Because the graph is non-cyclic the model implementation does not need a graph copy when calculating velocities and positions for the next step forward in time.  It may simply iterate graph-nodes in-place such that when a point-mass is moved in the modelled physical space no other point-mass depends on its physical-space position.
In each such time-step the springs with respective K factors yields modifications on the point-masses velocities.
The user-interface application adds a border condition that lightly bounces a mass off the frame.

# WebAssembly Interface

The web of point-masses are edited by *adding a dot* and scrolling the mouse-wheel (or clicking an up/down triangle) in order to define its mass.
The mass M at a given dot is indicated by the area of the drawn dot.
When a dot is added *and only at that time* it may be *connected* to the existing dots.
In this way the connected dots forms a graph already non-cyclic
as created dot by dot, and the web can be passed as-is to the spring-model.
When a dot is connected a line is drawn to it, representing a spring.
The K factor of the spring is adjusted by the mouse-wheel (or alternatively by the up/down triangles).
Clicking a second time on a dot will *remove* last added *either dot or line*.

When the web drawn is satisfactory to the observer,
the model may be *run* by clicking the right-pointing triangle on the top left.
Clicking anywhere once running will displace the last (the selected) point-mass added.
Any other point-mass can be selected by the mouse-wheel (or clicking the arrow triangles).
Clicking the upper left double-rectangle (swapped for the triangle to run)
will go back to edit-mode so that additional dots and lines may be added,
or the existing ones removed or the last objects respective K or M value modified.

http://rmotd.net/spring/

# Bugs
Instability *sometimes* occurs after a while.
It seems that both the spring model and border conditions are correct.
Therefore the probable source of instabilities is that *numerical rounding*
are accumulated in resonance with the springs.
Some dampening like friction in the model could be made
to overcome such amplification of numerical mishaps.
This would make the animation less entertaining.
Another usage of the model (usable independently of this application)
could add modeling of such additional physics in the same manner that
this application implements a bouncing border.

# Implementation Details
The reason the *inverse* and not the plain value of the mass is kept for the respective point-mass objects is merely to omit a division in the calculations of accelerations.  Probably an insignificant optimization, which have not been profiled.  Also note that *no tests* comes along with this code, which was a result from trying out with the interface instead of by tests.  Under development a mistake was done where while creating point-masses referencing previous ones, they were *appended* to the set of point-masses, which gave re-allocation and stale data, so that parts of the web did not move when running the interface.  Another error that was made at one time during development was omitting to take the *root* on calculation of distance.  *Lame* mistakes.  Maybe rolling this using more in-code tests would have been *quicker after all*.
